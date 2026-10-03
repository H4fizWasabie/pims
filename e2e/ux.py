"""Browser regression check for the UX layer (toasts, dialogs, session expiry,
drafts, retry, order replay, XSS). Not part of CI: needs python3 playwright + chromium.

Run it against a scratch database (it wipes orders/sessions there):
  createdb pims_e2e
  PORT=18083 DATABASE_URL=postgres://pims@127.0.0.1:54329/pims_e2e?sslmode=disable \
    PROCURA_DB_PATH=/tmp/procura.sqlite ADMIN_EMAIL=boss@example.com \
    ADMIN_PASSWORD=e2epassword1 MASTER_ADMINS=boss@example.com go run . &
  python3 e2e/ux.py
(/tmp/procura.sqlite only needs an empty `items(stock_id, current_stock)` table; edit
the PSQL line below if your Postgres port/user differ.)
"""
import json, sys, time, sqlite3, subprocess
from playwright.sync_api import sync_playwright
U='http://127.0.0.1:18083'
PSQL=['/usr/lib/postgresql/%s/bin/psql'%subprocess.check_output('ls /usr/lib/postgresql',shell=True,text=True).split()[-1],'-h','127.0.0.1','-p','54329','-U','pims','-d','pims_e2e','-tA','-c']
def sql(q): return subprocess.check_output(PSQL+[q],text=True).strip()
res=[]
def check(name, ok, extra=''):
    res.append(ok); print(('PASS ' if ok else 'FAIL ')+name+(' '+str(extra) if extra and not ok else ''))
with sync_playwright() as p:
    b=p.chromium.launch(); ctx=b.new_context(); page=ctx.new_page()
    errors=[]; page.on('pageerror',lambda e: errors.append(str(e)))
    dialogs=[]; page.on('dialog',lambda d:(dialogs.append(d.message),d.dismiss()))   # native alert/confirm must never appear
    page.goto(U); page.wait_for_selector('#login_email')
    page.fill('#login_email','boss@example.com'); page.fill('#login_password','e2epassword1'); page.click('#login_btn')
    page.wait_for_selector('#login-overlay',state='hidden'); 
    check('login + dashboard loads', page.inner_text('#pageTitleDisplay')=='Dashboard')
    # --- lab order with hostile name
    page.click('#nav-lab_order')
    payload='<img src=x onerror=window.__xss=1> Hostile'
    page.evaluate("p=>lab_selectItem({itemName:p,stockId:'X1',uom:'EA',currentStock:0,cost:'2',supplier:''})",payload)
    page.fill('#lab_qtyInput','3'); page.click('button[onclick="lab_addItem()"]')
    check('cart shows literal text, no XSS', page.evaluate("window.__xss")is None and 'Hostile' in page.inner_text('#lab_cartTableBody') and '<img' in page.inner_text('#lab_cartTableBody'))
    draft=page.evaluate("Object.keys(localStorage).filter(k=>k.startsWith('pims_draft:boss@example.com:lab'))")
    check('draft saved to localStorage', len(draft)==1, draft)
    # --- reload -> login -> draft restored
    page.reload(); page.wait_for_selector('#toast-host',state='attached'); page.wait_for_timeout(1500)
    check('draft restored after reload', 'Hostile' in page.inner_text('#lab_cartTableBody') or page.evaluate("lab_cart.length")==1)
    check('restore toast shown', 'Restored your unsent work' in page.inner_text('#toast-host'))
    page.click('#nav-lab_order')
    # --- session expiry mid-work
    sql("UPDATE sessions SET expires_at = NOW() - INTERVAL '1 minute'")
    page.click('#nav-lab_order') if False else None
    page.evaluate("api('GET','/api/order/list').catch(()=>{})"); page.wait_for_timeout(800)
    notice=page.inner_text('#login_notice')
    check('expiry shows login with explanation', page.is_visible('#login-overlay') and 'session ended' in notice, notice)
    check('no blocking native dialog on expiry', dialogs==[], dialogs)
    page.fill('#login_email','boss@example.com'); page.fill('#login_password','e2epassword1'); page.click('#login_btn')
    page.wait_for_selector('#login-overlay',state='hidden'); page.wait_for_timeout(500)
    check('re-login resumes same tab with cart intact', page.evaluate("currentTab")=='lab_order' and page.evaluate("lab_cart.length")==1, page.evaluate("currentTab"))
    # --- submit with dialog confirm; double click safe
    page.click('#lab_btnSubmit'); page.wait_for_selector('#ux-dialog[open]')
    page.click('#ux-dialog .ux-ok'); page.wait_for_selector('.toast.success'); 
    t=page.inner_text('#toast-host'); check('order success toast with PRF number', 'PRF-' in t, t)
    check('cart cleared + draft removed', page.evaluate("lab_cart.length")==0 and page.evaluate("Object.keys(localStorage).filter(k=>k.includes(':lab')).length")==0)
    check('exactly one order row stored', sql("SELECT COUNT(DISTINCT prf_no)||'/'||COUNT(*) FROM orders WHERE item_name LIKE '%Hostile'")=='1/1')
    # --- cancel confirm keeps data
    page.evaluate("p=>lab_selectItem({itemName:p,stockId:'X2',uom:'EA',currentStock:0,cost:'1',supplier:''})",'Keep me'); page.fill('#lab_qtyInput','1'); page.click('button[onclick="lab_addItem()"]')
    page.click('#lab_btnSubmit'); page.wait_for_selector('#ux-dialog[open]'); page.click('#ux-dialog .ux-cancel'); page.wait_for_timeout(300)
    check('cancel confirm submits nothing', page.evaluate("lab_cart.length")==1 and sql("SELECT COUNT(*) FROM orders WHERE item_name='Keep me'")=='0')
    # --- GRN keeps data when switching tabs
    page.click('#nav-grn'); page.wait_for_timeout(500)
    page.evaluate("GRN_CART_ITEMS.push({itemName:'G',qtyPo:1,qtyDo:1,qtyInv:1,uom:'pc',batch:'',status:'Match',remarks:''}); grn_renderTable()")
    page.click('#nav-lab_order'); page.click('#nav-grn'); page.wait_for_timeout(500)
    check('GRN draft survives tab switch', page.evaluate("GRN_CART_ITEMS.length")==1)
    # --- idle warning (shrink timers)
    page.evaluate("IDLE_TIMEOUT_MS=4000; IDLE_WARN_MS=2000; resetIdleTimer()"); page.wait_for_timeout(2600)
    check('idle warning toast with Stay signed in', 'Stay signed in' in page.inner_text('#toast-host'))
    page.mouse.move(120,140); page.wait_for_timeout(400)
    check('activity dismisses the warning', 'Stay signed in' not in page.inner_text('#toast-host'))
    # real inactivity -> signed out with explanation, work kept
    page.evaluate("resetIdleTimer()"); page.wait_for_timeout(4600)
    n=page.inner_text('#login_notice')
    check('inactivity signs out with explanation', page.is_visible('#login-overlay') and 'inactivity' in n, n)
    page.fill('#login_email','boss@example.com'); page.fill('#login_password','e2epassword1'); page.click('#login_btn')
    page.wait_for_selector('#login-overlay',state='hidden'); page.wait_for_timeout(400)
    check('work kept after inactivity sign-out', page.evaluate("GRN_CART_ITEMS.length")==1 and page.evaluate("currentTab")=='grn')
    page.evaluate("IDLE_TIMEOUT_MS=600000; IDLE_WARN_MS=60000; resetIdleTimer()")
    # --- load failure -> inline error + Retry
    page.route('**/api/dashboard/summary', lambda r: r.abort())
    page.click('#nav-dashboard'); page.wait_for_selector('#dash_loader .load-error')
    check('dashboard error shows retry', 'Retry' in page.inner_text('#dash_loader'))
    page.unroute('**/api/dashboard/summary'); page.click('#dash_loader button'); page.wait_for_timeout(800)
    check('retry recovers', not page.is_visible('#dash_loader'))
    # --- lost response, then retry: no duplicate order
    page.click('#nav-lab_order')
    page.evaluate("p=>lab_selectItem({itemName:p,stockId:'X3',uom:'EA',currentStock:0,cost:'5',supplier:''})",'Retry item'); page.fill('#lab_qtyInput','2'); page.click('button[onclick="lab_addItem()"]')
    def drop_response(route):
        route.fetch(); route.abort()          # server saves the order, browser never sees the answer
    page.route('**/api/order/generate', drop_response)
    page.click('#lab_btnSubmit'); page.wait_for_selector('#ux-dialog[open]'); page.click('#ux-dialog .ux-ok')
    page.wait_for_selector('.toast.error'); 
    check('lost response shows an error and keeps the cart', page.evaluate("lab_cart.some(i=>i.itemName=='Retry item')"))
    page.unroute('**/api/order/generate')
    page.click('#lab_btnSubmit'); page.wait_for_selector('#ux-dialog[open]'); page.click('#ux-dialog .ux-ok'); page.wait_for_timeout(800)
    check('retry replays instead of duplicating', sql("SELECT COUNT(DISTINCT prf_no)||'/'||COUNT(*) FROM orders WHERE item_name='Retry item'")=='1/1', sql("SELECT COUNT(*) FROM orders WHERE item_name='Retry item'"))
    check('retry clears cart', page.evaluate("lab_cart.length")==0)
    # --- different user on the same browser never sees previous user's draft
    page.evaluate("p=>lab_selectItem({itemName:p,stockId:'X4',uom:'EA',currentStock:0,cost:'1',supplier:''})",'Boss private draft'); page.fill('#lab_qtyInput','1'); page.click('button[onclick="lab_addItem()"]')
    page.evaluate("api('POST','/api/users/create',{email:'clerk@example.com',password:'clerkpassword1',role:'user'})"); page.wait_for_timeout(500)
    page.evaluate("handleLogout()"); page.wait_for_selector('#login-overlay',state='visible')
    page.fill('#login_email','clerk@example.com'); page.fill('#login_password','clerkpassword1'); page.click('#login_btn')
    page.wait_for_load_state('load'); page.wait_for_selector('#login-overlay',state='hidden'); page.wait_for_timeout(800)
    check('other user gets a clean page', page.evaluate("lab_cart.length")==0 and 'Boss private' not in page.content())
    check('no JS errors', errors==[], errors)
    check('no native alert/confirm ever shown', dialogs==[], dialogs)
    b.close()
print('%d/%d passed'%(sum(res),len(res))); sys.exit(0 if all(res) else 1)
