const {
  useState,
  useEffect
} = React;

// --- Inline Icons using Lucide patterns ---
const IconCamera = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("path", {
  d: "M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"
}), /*#__PURE__*/React.createElement("circle", {
  cx: "12",
  cy: "13",
  r: "3"
}));
const IconCheck = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("path", {
  d: "M22 11.08V12a10 10 0 1 1-5.93-9.14"
}), /*#__PURE__*/React.createElement("polyline", {
  points: "22 4 12 14.01 9 11.01"
}));
const IconPlus = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "10"
}), /*#__PURE__*/React.createElement("line", {
  x1: "12",
  y1: "8",
  x2: "12",
  y2: "16"
}), /*#__PURE__*/React.createElement("line", {
  x1: "8",
  y1: "12",
  x2: "16",
  y2: "12"
}));
const IconRotate = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("path", {
  d: "M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"
}), /*#__PURE__*/React.createElement("path", {
  d: "M3 3v5h5"
}));
const IconSave = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("path", {
  d: "M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"
}), /*#__PURE__*/React.createElement("polyline", {
  points: "17 21 17 13 7 13 7 21"
}), /*#__PURE__*/React.createElement("polyline", {
  points: "7 3 7 8 15 8"
}));
const IconArrowLeft = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("line", {
  x1: "19",
  y1: "12",
  x2: "5",
  y2: "12"
}), /*#__PURE__*/React.createElement("polyline", {
  points: "12 19 5 12 12 5"
}));
const IconList = () => /*#__PURE__*/React.createElement("svg", {
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, /*#__PURE__*/React.createElement("line", {
  x1: "8",
  y1: "6",
  x2: "21",
  y2: "6"
}), /*#__PURE__*/React.createElement("line", {
  x1: "8",
  y1: "12",
  x2: "21",
  y2: "12"
}), /*#__PURE__*/React.createElement("line", {
  x1: "8",
  y1: "18",
  x2: "21",
  y2: "18"
}), /*#__PURE__*/React.createElement("line", {
  x1: "3",
  y1: "6",
  x2: "3.01",
  y2: "6"
}), /*#__PURE__*/React.createElement("line", {
  x1: "3",
  y1: "12",
  x2: "3.01",
  y2: "12"
}), /*#__PURE__*/React.createElement("line", {
  x1: "3",
  y1: "18",
  x2: "3.01",
  y2: "18"
}));

// Utility: Image resizing
const resizeImage = (base64Str, maxWidth = 1024) => {
  return new Promise(resolve => {
    const img = new Image();
    img.src = base64Str;
    img.onload = () => {
      const canvas = document.createElement('canvas');
      let width = img.width,
        height = img.height;
      if (width > maxWidth) {
        height *= maxWidth / width;
        width = maxWidth;
      }
      canvas.width = width;
      canvas.height = height;
      canvas.getContext('2d').drawImage(img, 0, 0, width, height);
      resolve(canvas.toDataURL('image/jpeg', 0.8));
    };
    img.onerror = () => resolve(base64Str);
  });
};

// --- Fuzzy Matching Utilities ---
const levenshtein = (a, b) => {
  const m = a.length,
    n = b.length;
  if (!m) return n;
  if (!n) return m;
  const d = Array.from({
    length: m + 1
  }, (_, i) => {
    const row = new Array(n + 1);
    row[0] = i;
    return row;
  });
  for (let j = 1; j <= n; j++) d[0][j] = j;
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      d[i][j] = a[i - 1] === b[j - 1] ? d[i - 1][j - 1] : 1 + Math.min(d[i - 1][j], d[i][j - 1], d[i - 1][j - 1]);
    }
  }
  return d[m][n];
};
const normalize = s => String(s || '').toLowerCase().replace(/[^a-z0-9]/g, '');
const scoreMatch = (query, candidate) => {
  const q = normalize(query),
    c = normalize(candidate);
  if (!q || !c) return 0;
  // Normalized edit distance (0 = identical, 1 = completely different)
  const maxLen = Math.max(q.length, c.length);
  const editScore = 1 - levenshtein(q, c) / maxLen;
  // Substring bonus
  const subBonus = c.includes(q) || q.includes(c) ? 0.25 : 0;
  // Word-start bonus: if first 3 chars match
  const startBonus = q.slice(0, 3) === c.slice(0, 3) ? 0.15 : 0;
  return Math.min(1, editScore + subBonus + startBonus);
};
const getTopMatches = (query, masterData, limit = 5) => {
  if (!query || !masterData || masterData.length === 0) return [];
  return masterData.map(item => ({
    ...item,
    score: scoreMatch(query, item.itemName)
  })).filter(item => item.score > 0.25).sort((a, b) => b.score - a.score).slice(0, limit);
};

// Shared Input Component
const InputField = ({
  label,
  type = "text",
  ...props
}) => /*#__PURE__*/React.createElement("div", {
  style: {
    marginBottom: '12px'
  }
}, /*#__PURE__*/React.createElement("label", {
  style: {
    display: 'block',
    fontSize: '12px',
    color: '#64748b',
    marginBottom: '4px',
    fontWeight: '600',
    textTransform: 'uppercase'
  }
}, label), /*#__PURE__*/React.createElement("input", {
  type: type,
  style: {
    width: '100%',
    padding: '12px',
    borderRadius: '8px',
    border: '1px solid #e2e8f0',
    backgroundColor: '#f8fafc',
    outline: 'none',
    transition: 'box-shadow 0.2s',
    fontSize: '15px'
  },
  onFocus: e => e.target.style.boxShadow = '0 0 0 2px rgba(79, 70, 229, 0.2)',
  onBlur: e => e.target.style.boxShadow = 'none',
  ...props
}));

// SCREEN 1: Loading
const LoadingScreen = ({
  message
}) => /*#__PURE__*/React.createElement("div", {
  style: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    padding: '40px'
  }
}, /*#__PURE__*/React.createElement("i", {
  className: "fa-solid fa-spinner fa-spin",
  style: {
    fontSize: '48px',
    color: '#4f46e5',
    marginBottom: '16px'
  }
}), /*#__PURE__*/React.createElement("p", {
  style: {
    fontSize: '18px',
    fontWeight: 'bold',
    color: '#1e293b'
  }
}, message || 'Processing...'), /*#__PURE__*/React.createElement("p", {
  style: {
    fontSize: '14px',
    color: '#64748b',
    marginTop: '8px'
  }
}, "AI is extracting product details"));

// SCREEN 2: Camera Capture
const CameraScreen = ({
  onCaptureComplete
}) => {
  const [images, setImages] = useState([]);
  const handleFile = async e => {
    const file = e.target.files[0];
    e.target.value = '';
    if (!file) return;
    const reader = new FileReader();
    reader.onloadend = async () => {
      const resized = await resizeImage(reader.result);
      setImages(prev => [...prev, resized]);
    };
    reader.readAsDataURL(file);
  };
  if (images.length > 0) {
    const currentImg = images[images.length - 1];
    return /*#__PURE__*/React.createElement("div", {
      style: {
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        backgroundColor: '#000'
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        flex: '1',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'relative',
        overflow: 'hidden'
      }
    }, /*#__PURE__*/React.createElement("img", {
      src: currentImg,
      style: {
        width: '100%',
        height: '100%',
        objectFit: 'contain'
      }
    }), /*#__PURE__*/React.createElement("div", {
      style: {
        position: 'absolute',
        top: '20px',
        right: '20px',
        backgroundColor: 'rgba(0,0,0,0.6)',
        color: 'white',
        padding: '6px 12px',
        borderRadius: '20px',
        fontSize: '12px',
        fontWeight: 'bold'
      }
    }, "Photo ", images.length)), /*#__PURE__*/React.createElement("div", {
      style: {
        backgroundColor: '#1e293b',
        padding: '24px',
        borderTopLeftRadius: '24px',
        borderTopRightRadius: '24px'
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        display: 'flex',
        gap: '12px',
        marginBottom: '16px'
      }
    }, /*#__PURE__*/React.createElement("button", {
      onClick: () => setImages(prev => prev.slice(0, -1)),
      style: {
        flex: 1,
        padding: '12px',
        borderRadius: '12px',
        backgroundColor: '#334155',
        color: 'white',
        border: '1px solid #475569',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        fontWeight: 'bold'
      }
    }, /*#__PURE__*/React.createElement(IconRotate, null), " Retake"), /*#__PURE__*/React.createElement("label", {
      style: {
        flex: 1,
        padding: '12px',
        borderRadius: '12px',
        backgroundColor: '#334155',
        color: '#60a5fa',
        border: '1px solid #475569',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        fontWeight: 'bold',
        cursor: 'pointer',
        margin: 0
      }
    }, /*#__PURE__*/React.createElement("input", {
      type: "file",
      accept: "image/*",
      capture: "environment",
      style: {
        display: 'none'
      },
      onChange: handleFile
    }), /*#__PURE__*/React.createElement(IconPlus, null), " Add Angle")), /*#__PURE__*/React.createElement("button", {
      onClick: () => onCaptureComplete(images),
      style: {
        width: '100%',
        padding: '16px',
        borderRadius: '12px',
        backgroundColor: '#4f46e5',
        color: 'white',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        fontWeight: 'bold',
        fontSize: '18px',
        boxShadow: '0 4px 14px rgba(79, 70, 229, 0.4)',
        border: 'none'
      }
    }, /*#__PURE__*/React.createElement(IconCheck, null), " Process ", images.length > 1 ? `${images.length} Photos` : 'Photo')));
  }
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      backgroundColor: '#f8fafc',
      padding: '24px'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      marginBottom: '32px'
    }
  }, /*#__PURE__*/React.createElement("h1", {
    style: {
      fontSize: '28px',
      fontWeight: '800',
      color: '#0f172a',
      margin: '0 0 8px 0'
    }
  }, "Stock Take ", /*#__PURE__*/React.createElement("span", {
    style: {
      color: '#4f46e5'
    }
  }, "Scan")), /*#__PURE__*/React.createElement("p", {
    style: {
      color: '#64748b',
      margin: 0
    }
  }, "Take photos of product labels, batches, and expiry dates.")), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center'
    }
  }, /*#__PURE__*/React.createElement("label", {
    style: {
      width: '100%',
      maxWidth: '320px',
      aspectRatio: '3/4',
      backgroundColor: 'white',
      border: '3px dashed #cbd5e1',
      borderRadius: '32px',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      cursor: 'pointer',
      boxShadow: '0 10px 25px rgba(0,0,0,0.05)',
      transition: 'transform 0.2s'
    },
    onMouseDown: e => e.currentTarget.style.transform = 'scale(0.96)',
    onMouseUp: e => e.currentTarget.style.transform = 'scale(1)'
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      padding: '20px',
      backgroundColor: '#e0e7ff',
      borderRadius: '50%',
      marginBottom: '24px'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      color: '#4f46e5'
    }
  }, /*#__PURE__*/React.createElement(IconCamera, null))), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '20px',
      fontWeight: 'bold',
      color: '#1e293b'
    }
  }, "Tap to Scan"), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '13px',
      color: '#94a3b8',
      marginTop: '8px'
    }
  }, "Capture label & expiry side"), /*#__PURE__*/React.createElement("input", {
    type: "file",
    accept: "image/*",
    capture: "environment",
    style: {
      display: 'none'
    },
    onChange: handleFile
  }))));
};

// SCREEN 3: Review Form
const ReviewScreen = ({
  initialData,
  onSave,
  onBack,
  locations,
  masterData
}) => {
  const [data, setData] = useState({
    itemName: initialData.itemName || '',
    batch: initialData.batch || '',
    expiry: initialData.expiry || '',
    qty: '',
    location: localStorage.getItem('take_lastLoc') || '',
    stockId: initialData.stockId || '',
    uom: initialData.uom || '',
    cost: initialData.cost || 0
  });

  // Compute fuzzy matches against the AI-scanned name
  const fuzzyMatches = React.useMemo(() => getTopMatches(initialData.itemName, masterData), [initialData.itemName, masterData]);
  const [selectedMatchIdx, setSelectedMatchIdx] = useState(null);
  const [manualSearch, setManualSearch] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);

  // Live fuzzy results as user types in the Final Product Name field
  const manualSearchResults = React.useMemo(() => manualSearch.length >= 2 ? getTopMatches(manualSearch, masterData, 8) : [], [manualSearch, masterData]);
  const selectManualMatch = match => {
    setData(prev => ({
      ...prev,
      stockId: match.stockId,
      itemName: match.itemName,
      uom: match.uom || prev.uom,
      cost: match.cost || prev.cost
    }));
    setManualSearch('');
    setShowSearchResults(false);
  };

  // Auto-select best match on mount if confidence is very high
  useEffect(() => {
    if (fuzzyMatches.length > 0 && fuzzyMatches[0].score >= 0.7 && selectedMatchIdx === null) {
      selectMatch(0);
    }
  }, [fuzzyMatches]);
  const selectMatch = idx => {
    const match = fuzzyMatches[idx];
    setSelectedMatchIdx(idx);
    setData(prev => ({
      ...prev,
      stockId: match.stockId,
      itemName: match.itemName,
      uom: match.uom || prev.uom,
      cost: match.cost || prev.cost
    }));
  };
  const handleChange = (field, value) => setData(prev => ({
    ...prev,
    [field]: value
  }));
  const save = () => {
    if (!data.location) return alert('Please select a location');
    if (!data.itemName) return alert('Product Name is required');
    if (!data.qty) return alert('Physical quantity is required');
    localStorage.setItem('take_lastLoc', data.location);
    onSave(data);
  };
  const confidenceColor = score => {
    if (score >= 0.7) return {
      bg: '#dcfce7',
      text: '#166534'
    };
    if (score >= 0.5) return {
      bg: '#fef9c3',
      text: '#854d0e'
    };
    return {
      bg: '#fee2e2',
      text: '#991b1b'
    };
  };
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      backgroundColor: '#f1f5f9'
    }
  }, /*#__PURE__*/React.createElement("header", {
    style: {
      backgroundColor: 'white',
      padding: '16px 20px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      borderBottom: '1px solid #e2e8f0',
      zIndex: 10
    }
  }, /*#__PURE__*/React.createElement("button", {
    onClick: onBack,
    style: {
      background: 'none',
      border: 'none',
      color: '#64748b',
      cursor: 'pointer'
    }
  }, /*#__PURE__*/React.createElement(IconArrowLeft, null)), /*#__PURE__*/React.createElement("h2", {
    style: {
      fontSize: '18px',
      fontWeight: '700',
      margin: 0,
      color: '#0f172a'
    }
  }, "Review & Count"), /*#__PURE__*/React.createElement("div", {
    style: {
      width: '24px'
    }
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      overflowY: 'auto',
      padding: '20px'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      backgroundColor: 'white',
      borderRadius: '16px',
      padding: '20px',
      marginBottom: '16px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.02)'
    }
  }, /*#__PURE__*/React.createElement("h3", {
    style: {
      fontSize: '13px',
      fontWeight: '700',
      color: '#94a3b8',
      textTransform: 'uppercase',
      marginBottom: '16px',
      letterSpacing: '0.5px'
    }
  }, "AI Extraction"), /*#__PURE__*/React.createElement(InputField, {
    label: "Scanned Product Name",
    value: initialData.itemName || '',
    readOnly: true,
    style: {
      backgroundColor: '#f1f5f9',
      color: '#64748b'
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: '12px'
    }
  }, /*#__PURE__*/React.createElement(InputField, {
    label: "Batch No.",
    value: data.batch,
    onChange: e => handleChange('batch', e.target.value)
  }), /*#__PURE__*/React.createElement(InputField, {
    label: "Expiry (MM/YYYY)",
    value: data.expiry,
    onChange: e => handleChange('expiry', e.target.value)
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      backgroundColor: 'white',
      borderRadius: '16px',
      padding: '20px',
      marginBottom: '16px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.02)'
    }
  }, /*#__PURE__*/React.createElement("h3", {
    style: {
      fontSize: '13px',
      fontWeight: '700',
      color: '#94a3b8',
      textTransform: 'uppercase',
      marginBottom: '12px',
      letterSpacing: '0.5px'
    }
  }, "Matched Product"), selectedMatchIdx !== null && /*#__PURE__*/React.createElement("div", {
    style: {
      padding: '10px 14px',
      backgroundColor: '#eef2ff',
      borderRadius: '10px',
      marginBottom: '12px',
      border: '2px solid #4f46e5'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: '11px',
      color: '#4f46e5',
      fontWeight: '700',
      marginBottom: '2px'
    }
  }, "STOCK ID: ", data.stockId), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: '15px',
      fontWeight: '700',
      color: '#1e293b'
    }
  }, data.itemName), data.uom && /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: '12px',
      color: '#64748b',
      marginTop: '2px'
    }
  }, "UOM: ", data.uom, " · Cost: ", data.cost)), fuzzyMatches.length > 0 ? /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: '6px'
    }
  }, /*#__PURE__*/React.createElement("label", {
    style: {
      fontSize: '11px',
      color: '#94a3b8',
      fontWeight: '600'
    }
  }, "TAP TO SELECT A MATCH:"), fuzzyMatches.map((match, idx) => {
    const cc = confidenceColor(match.score);
    const isSelected = selectedMatchIdx === idx;
    return /*#__PURE__*/React.createElement("button", {
      key: match.stockId + idx,
      onClick: () => selectMatch(idx),
      style: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px 14px',
        borderRadius: '10px',
        border: isSelected ? '2px solid #4f46e5' : '1px solid #e2e8f0',
        backgroundColor: isSelected ? '#eef2ff' : '#f8fafc',
        cursor: 'pointer',
        textAlign: 'left',
        transition: 'all 0.15s'
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        flex: 1,
        minWidth: 0
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        fontSize: '11px',
        color: '#4f46e5',
        fontWeight: '700'
      }
    }, match.stockId), /*#__PURE__*/React.createElement("div", {
      style: {
        fontSize: '14px',
        fontWeight: '600',
        color: '#1e293b',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap'
      }
    }, match.itemName)), /*#__PURE__*/React.createElement("span", {
      style: {
        fontSize: '11px',
        fontWeight: '700',
        padding: '3px 8px',
        borderRadius: '6px',
        backgroundColor: cc.bg,
        color: cc.text,
        flexShrink: 0,
        marginLeft: '8px'
      }
    }, Math.round(match.score * 100), "%"));
  })) : /*#__PURE__*/React.createElement("div", {
    style: {
      padding: '16px',
      textAlign: 'center',
      color: '#94a3b8',
      fontSize: '13px',
      backgroundColor: '#fef2f2',
      borderRadius: '10px'
    }
  }, "No match found in master data — manual entry below"), /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: '12px',
      position: 'relative'
    }
  }, /*#__PURE__*/React.createElement("label", {
    style: {
      display: 'block',
      fontSize: '12px',
      color: '#64748b',
      marginBottom: '4px',
      fontWeight: '600',
      textTransform: 'uppercase'
    }
  }, "Final Product Name (search)"), /*#__PURE__*/React.createElement("input", {
    type: "text",
    value: data.itemName,
    onChange: e => {
      handleChange('itemName', e.target.value);
      setSelectedMatchIdx(null);
      setManualSearch(e.target.value);
      setShowSearchResults(true);
    },
    placeholder: "Type to search master data...",
    style: {
      width: '100%',
      padding: '12px',
      borderRadius: '8px',
      border: '1px solid #e2e8f0',
      backgroundColor: '#f8fafc',
      outline: 'none',
      transition: 'box-shadow 0.2s',
      fontSize: '15px'
    },
    onFocus: e => {
      e.target.style.boxShadow = '0 0 0 2px rgba(79, 70, 229, 0.2)';
      if (data.itemName) {
        setManualSearch(data.itemName);
        setShowSearchResults(true);
      }
    },
    onBlur: e => {
      e.target.style.boxShadow = 'none';
      setTimeout(() => setShowSearchResults(false), 200);
    }
  }), showSearchResults && data.itemName && manualSearchResults.length > 0 && /*#__PURE__*/React.createElement("div", {
    style: {
      position: 'absolute',
      top: '100%',
      left: 0,
      right: 0,
      zIndex: 50,
      backgroundColor: 'white',
      border: '1px solid #e2e8f0',
      borderRadius: '10px',
      boxShadow: '0 8px 24px rgba(0,0,0,0.12)',
      maxHeight: '200px',
      overflowY: 'auto',
      marginTop: '4px'
    }
  }, manualSearchResults.map((match, idx) => {
    const cc = confidenceColor(match.score);
    return /*#__PURE__*/React.createElement("button", {
      key: match.stockId + '-s-' + idx,
      onMouseDown: e => {
        e.preventDefault();
        selectManualMatch(match);
      },
      style: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        width: '100%',
        padding: '10px 14px',
        border: 'none',
        borderBottom: '1px solid #f1f5f9',
        backgroundColor: 'white',
        cursor: 'pointer',
        textAlign: 'left'
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        flex: 1,
        minWidth: 0
      }
    }, /*#__PURE__*/React.createElement("div", {
      style: {
        fontSize: '11px',
        color: '#4f46e5',
        fontWeight: '700'
      }
    }, match.stockId), /*#__PURE__*/React.createElement("div", {
      style: {
        fontSize: '13px',
        fontWeight: '600',
        color: '#1e293b',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap'
      }
    }, match.itemName)), /*#__PURE__*/React.createElement("span", {
      style: {
        fontSize: '10px',
        fontWeight: '700',
        padding: '2px 6px',
        borderRadius: '4px',
        backgroundColor: cc.bg,
        color: cc.text,
        flexShrink: 0,
        marginLeft: '6px'
      }
    }, Math.round(match.score * 100), "%"));
  })), data.stockId && /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: '6px',
      fontSize: '12px',
      color: '#4f46e5',
      fontWeight: '600'
    }
  }, "✓ Stock ID: ", data.stockId))), /*#__PURE__*/React.createElement("div", {
    style: {
      backgroundColor: 'white',
      borderRadius: '16px',
      padding: '20px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.02)'
    }
  }, /*#__PURE__*/React.createElement("h3", {
    style: {
      fontSize: '13px',
      fontWeight: '700',
      color: '#94a3b8',
      textTransform: 'uppercase',
      marginBottom: '16px',
      letterSpacing: '0.5px'
    }
  }, "Physical Count"), /*#__PURE__*/React.createElement("div", {
    style: {
      marginBottom: '12px'
    }
  }, /*#__PURE__*/React.createElement("label", {
    style: {
      display: 'block',
      fontSize: '12px',
      color: '#64748b',
      marginBottom: '4px',
      fontWeight: '600',
      textTransform: 'uppercase'
    }
  }, "Location"), /*#__PURE__*/React.createElement("select", {
    value: data.location,
    onChange: e => handleChange('location', e.target.value),
    style: {
      width: '100%',
      padding: '12px',
      borderRadius: '8px',
      border: '1px solid #e2e8f0',
      backgroundColor: '#eef2ff',
      color: '#312e81',
      fontWeight: 'bold',
      outline: 'none',
      fontSize: '15px'
    }
  }, /*#__PURE__*/React.createElement("option", {
    value: "",
    disabled: true
  }, "Select Location"), locations.map(loc => /*#__PURE__*/React.createElement("option", {
    key: loc,
    value: loc
  }, loc)))), /*#__PURE__*/React.createElement(InputField, {
    label: "Physical Quantity",
    type: "number",
    value: data.qty,
    onChange: e => handleChange('qty', e.target.value),
    placeholder: "0"
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      backgroundColor: 'white',
      padding: '20px',
      borderTop: '1px solid #e2e8f0'
    }
  }, /*#__PURE__*/React.createElement("button", {
    onClick: save,
    style: {
      width: '100%',
      padding: '16px',
      borderRadius: '12px',
      backgroundColor: '#4f46e5',
      color: 'white',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: '8px',
      fontWeight: 'bold',
      fontSize: '18px',
      boxShadow: '0 4px 14px rgba(79, 70, 229, 0.4)',
      border: 'none',
      cursor: 'pointer'
    }
  }, /*#__PURE__*/React.createElement(IconSave, null), " Record Count")));
};

// MAIN APP COMPONENT
const StockTakeApp = () => {
  const [view, setView] = useState('camera'); // camera | processing | review | saving
  const [images, setImages] = useState([]);
  const [extractedData, setExtractedData] = useState({});
  const [masterData, setMasterData] = useState([]);
  const locations = ["Pharmacy Level 1", "Lab Level 1", "Mini Pharmacy Level 3", "Lab Level 2", "Store Level 2", "Physiotherapy room"];

  // Fetch master data on load to match IDs later
  useEffect(() => {
    api("GET", "/api/master/all").then(data => {
      if (data) {
        // We need to filter out 'unavailable'/'not-available' items at the frontend
        // since getAllMasterItems returns them if we use a different endpoint or if we want to be safe.
        // Wait, getAllMasterItems is in gsorder.js and it already filters them!
        // But just in case Stock Take uses a different status field or we need to be absolutely sure:
        const availableData = data.filter(item => {
          if (!item.status) return true; // If status wasn't fetched, assume available
          const s = String(item.status).trim().toLowerCase();
          return s !== 'unavailable' && s !== 'not-available';
        });
        setMasterData(availableData);
      }
    }).catch(err => {
      console.error("Failed:", err);
    });
  }, []);
  const processImages = imageArray => {
    setView('processing');
    setImages(imageArray);

    // We pass all images to the backend to extract everything in one prompt
    // Extract base64 properly
    const b64Array = imageArray.map(img => img.split(',')[1]);
    api("POST", "/api/stocktake/analyze-image", {
      images: b64Array
    }).then(result => {
      if (result.error) {
        alert("AI Error: " + result.error);
      }

      // Pass raw AI result directly — fuzzy matching happens in ReviewScreen
      setExtractedData({
        itemName: result.productName || '',
        batch: result.batchNumber || '',
        expiry: result.expiryDate || ''
      });
      setView('review');
    }).catch(err => {
      alert("Extraction failed: " + err);
      setView('camera');
    });
  };
  const saveRecord = finalData => {
    setView('saving');
    // Clean up missing/empty fields
    const payload = {
      location: finalData.location,
      stockId: finalData.stockId || 'AI-SCAN',
      itemName: finalData.itemName || 'Unknown Item',
      uom: finalData.uom || 'N/A',
      group: 'N/A',
      cost: finalData.cost || 0,
      qty: parseFloat(finalData.qty) || 0,
      batch: finalData.batch || '',
      expiry: finalData.expiry || ''
    };
    api("POST", "/api/stocktake/submit", payload).then(() => {
      // Success! Give momentary feedback then back to camera
      setView('camera');
      setImages([]);
      setExtractedData({});
    }).catch(err => {
      alert("Error saving: " + err.message);
      setView('review');
    });
  };
  if (view === 'processing') return /*#__PURE__*/React.createElement(LoadingScreen, {
    message: "Extracting details..."
  });
  if (view === 'saving') return /*#__PURE__*/React.createElement(LoadingScreen, {
    message: "Saving stock count..."
  });
  if (view === 'review') return /*#__PURE__*/React.createElement(ReviewScreen, {
    initialData: extractedData,
    onSave: saveRecord,
    onBack: () => setView('camera'),
    locations: locations,
    masterData: masterData
  });
  return /*#__PURE__*/React.createElement(CameraScreen, {
    onCaptureComplete: processImages
  });
};
window.loadStockTake = function () {
  const rootEl = document.getElementById('stock-take-root');
  if (rootEl && !rootEl._stockTakeRoot) {
    rootEl._stockTakeRoot = ReactDOM.createRoot(rootEl);
    rootEl._stockTakeRoot.render(/*#__PURE__*/React.createElement(StockTakeApp, null));
  }
};
