"""Browser regression check for the UX layer (toasts, dialogs, session expiry,
drafts, retry, order replay, XSS). Not part of CI: needs python3 playwright + chromium.

Run it against a scratch database (it wipes orders/sessions there):
  createdb pims_e2e
  PORT=18083 DATABASE_URL=postgres://pims@127.0.0.1:54329/pims_e2e?sslmode=disable \
    PROCURA_DB_PATH=/tmp/procura.sqlite ADMIN_EMAIL=boss@example.com \
    ADMIN_PASSWORD=e2epassword1 MASTER_ADMINS=boss@example.com \
    INDENT_APPROVERS=boss@example.com SPEC_APPROVERS=boss@example.com go run . &
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
    external=[]; page.on('request',lambda r: external.append(r.url) if not r.url.startswith(U) and not r.url.startswith('data:') else None)
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
    # --- expiry filters + search
    sql("DELETE FROM expiry_tracking")
    for sid,name,days in [('E1','Syringe 5ml',-3),('E2','Gauze Roll',12),('E3','Suture 3-0',60),('E4','Vaccine Vial',150),('E5','Glove M',300)]:
        sql("INSERT INTO expiry_tracking (stock_id,item_name,batch_no,expiry_date) VALUES ('%s','%s','B-%s',CURRENT_DATE+%d)"%(sid,name,sid,days))
    page.click('#nav-expiry_tracking'); page.wait_for_selector('.exp-item-row'); page.wait_for_timeout(300)
    chips={c.get_attribute('data-band') or 'all': c.query_selector('.n').inner_text() for c in page.query_selector_all('#exp_chips .exp-chip')}
    check('chips show counts per level', chips=={'all':'5','expired':'1','critical':'1','action':'1','warning':'1','alert':'1'}, chips)
    page.click('#exp_chips [data-band="expired"]'); page.wait_for_timeout(500)
    rows=page.query_selector_all('.exp-item-row')
    check('expired chip filters the list', len(rows)==1 and 'Syringe' in rows[0].inner_text())
    check('active chip is aria-pressed', page.get_attribute('#exp_chips [data-band="expired"]','aria-pressed')=='true')
    page.click('#exp_chips [data-band=""]'); page.fill('#exp_search','glove'); page.wait_for_timeout(800)
    rows=page.query_selector_all('.exp-item-row')
    check('search narrows (debounced)', len(rows)==1 and 'Glove' in rows[0].inner_text())
    page.fill('#exp_search','zzzz'); page.wait_for_timeout(800)
    check('no-match message', 'No batches match' in page.inner_text('#exp_list'))
    page.fill('#exp_search','b-e3'); page.wait_for_timeout(800)
    check('search by batch', 'Suture' in page.inner_text('#exp_list'))
    page.fill('#exp_search',''); page.wait_for_timeout(600)
    page.route('**/api/expiry/list**', lambda r: r.abort())
    page.click('#exp_chips [data-band="critical"]'); page.wait_for_selector('#exp_list .load-error')
    page.unroute('**/api/expiry/list**'); page.click('#exp_list .load-error button'); page.wait_for_timeout(700)
    check('expiry error has working Retry', 'Gauze' in page.inner_text('#exp_list'))
    page.click('#exp_chips [data-band=""]'); page.wait_for_timeout(400)
    # --- inline validation
    page.click('#nav-lab_order')
    page.evaluate("lab_cart=[]; lab_renderCart(); lab_selectItem({itemName:'V',stockId:'V1',uom:'EA',currentStock:0,cost:'1',supplier:''})")
    page.fill('#lab_qtyInput',''); page.click('button[onclick="lab_addItem()"]'); page.wait_for_timeout(200)
    msg=page.inner_text('#lab_qtyInput + .field-msg')
    check('empty qty: message under the field', 'above 0' in msg and page.get_attribute('#lab_qtyInput','aria-invalid')=='true', msg)
    check('invalid field is focused', page.evaluate("document.activeElement.id")=='lab_qtyInput')
    page.fill('#lab_qtyInput','2'); page.wait_for_timeout(100)
    check('message clears when user edits', page.query_selector('#lab_qtyInput + .field-msg') is None and page.get_attribute('#lab_qtyInput','aria-invalid') is None)
    page.evaluate("document.getElementById('lab_stockId').value=''"); page.click('button[onclick="lab_addItem()"]'); page.wait_for_timeout(200)
    check('no item picked: message under search box', 'pick it' in page.inner_text('#lab_searchInput + .field-msg'))
    page.click('#nav-specification_form'); page.wait_for_timeout(300)
    page.click('#spec_btnSubmit'); page.wait_for_timeout(300)
    n=len(page.query_selector_all('#module-specification_form .field-msg'))
    check('spec form lists every missing field at once', n>=4, n)
    page.click('#nav-grn'); page.wait_for_timeout(500)
    page.evaluate("grn_addItem()") if False else None
    page.click('button[onclick="grn_addItem()"]') if page.query_selector('button[onclick="grn_addItem()"]') else None
    page.wait_for_timeout(200)
    check('grn add with nothing shows field messages', len(page.query_selector_all('#module-grn .field-msg'))>=1)
    # --- approver gating + bulk approve (currently signed in as the clerk)
    sql("DELETE FROM indents")
    for sid,stock,qty in [('BA',50,5),('BB',50,5),('BC',1,5)]:
        sql("INSERT INTO master_items (stock_id,item_name) VALUES ('%s','Bulk %s') ON CONFLICT DO NOTHING"%(sid,sid))
        pc=sqlite3.connect('/tmp/procura.sqlite'); pc.execute('DELETE FROM items WHERE stock_id=?',(sid,)); pc.execute('INSERT INTO items(stock_id,current_stock) VALUES (?,?)',(sid,stock)); pc.commit(); pc.close()
        sql("INSERT INTO indents (indent_id,requester,item_name,stock_id,uom,requested_qty) VALUES ('REQ-T','Lab','Bulk %s','%s','pc',%d)"%(sid,sid,qty))
    page.click('#nav-dashboard'); page.wait_for_selector('#dash_approvalBody tr'); page.wait_for_timeout(300)
    check('non-approver does not see approve controls', not page.is_visible('#dash_approvalBody .btn-approve') and not page.is_visible('#dash_selAll'))
    page.evaluate("handleLogout()"); page.wait_for_selector('#login-overlay',state='visible')
    page.fill('#login_email','boss@example.com'); page.fill('#login_password','e2epassword1'); page.click('#login_btn')
    page.wait_for_load_state('load'); page.wait_for_selector('#login-overlay',state='hidden'); page.wait_for_selector('#dash_approvalBody tr'); page.wait_for_timeout(400)
    check('approver sees controls', page.is_visible('#dash_approvalBody .btn-approve') and page.is_visible('#dash_selAll'))
    check('bulk bar hidden until something is selected', not page.is_visible('#dash_bulkBar'))
    page.click('#dash_selAll'); page.wait_for_timeout(100)
    check('select-all shows count', '3 selected' in page.inner_text('#dash_bulkCount'))
    pend_before=page.inner_text('#dash_valPending')
    page.click('#dash_bulkBar .btn-approve'); page.wait_for_selector('#ux-dialog[open]')
    check('bulk confirm names the count', '3 request' in page.inner_text('#ux-msg'))
    page.click('#ux-dialog .ux-ok'); page.wait_for_selector('.toast.warn'); page.wait_for_timeout(300)
    t=page.inner_text('#toast-host'); check('partial result toast', '2 approved' in t and '1 could not' in t, t)
    left=page.query_selector_all('#dash_approvalBody tr')
    check('approved rows leave, failed row stays with reason', len(left)==1 and 'insufficient' in left[0].inner_text().lower(), [l.inner_text() for l in left])
    check('pending KPI follows the table', page.inner_text('#dash_valPending')=='1', (pend_before, page.inner_text('#dash_valPending')))
    check('db: 2 approved, 1 pending', sql("SELECT COUNT(*) FILTER (WHERE status='Approved')||'/'||COUNT(*) FILTER (WHERE status='Pending') FROM indents")=='2/1')
    # --- accessibility
    check('pinch-zoom allowed', 'user-scalable' not in page.get_attribute('meta[name=viewport]','content') and page.get_attribute('html','lang')=='en')
    unnamed=page.evaluate("""Array.from(document.querySelectorAll('input,select,textarea,button')).filter(c=>c.type!=='hidden' && !c.disabled
        && !(c.getAttribute('aria-label')||c.getAttribute('aria-labelledby')||c.title||(c.labels&&c.labels.length)||(c.tagName==='BUTTON'&&c.textContent.trim())||(c.tagName==='SELECT'&&false))).map(c=>c.tagName+'#'+c.id+'.'+c.className)""")
    check('every control has an accessible name', unnamed==[], unnamed[:8])
    page.focus('#nav-dashboard'); page.keyboard.press('Tab'); page.keyboard.press('Tab')
    focused=page.evaluate("document.activeElement.id")
    page.keyboard.press('Enter'); page.wait_for_timeout(400)
    tab=page.evaluate("currentTab")
    check('nav links are keyboard reachable and operable', focused.startswith('nav-') and tab==focused[4:], (focused,tab))
    check('active nav exposes aria-current', page.get_attribute('#nav-'+tab,'aria-current')=='page')
    page.click('#nav-lab_order'); page.fill('#lab_searchInput','zzqx'); page.wait_for_timeout(1200)
    check('add-custom suggestion is focusable', page.evaluate("(document.querySelector('#lab_searchResults [onclick*=addCustom]')||{}).tabIndex")==0)
    # --- everything first-party
    check('no third-party requests', external==[], sorted(set(external))[:5])
    check('icon font loaded from our server', page.evaluate("document.fonts.check('900 1em \"Font Awesome 6 Free\"')") is True)
    check('no JS errors', errors==[], errors)
    check('no native alert/confirm ever shown', dialogs==[], dialogs)
    b.close()
print('%d/%d passed'%(sum(res),len(res))); sys.exit(0 if all(res) else 1)
