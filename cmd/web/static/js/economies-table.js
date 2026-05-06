(function () {
  var table = document.getElementById('economies-table');
  if (!table) {
    return;
  }

  var tbody = table.querySelector('tbody');
  if (!tbody) {
    return;
  }

  var sortAttr = {
    name: 'data-sort-name',
    domains: 'data-sort-domains',
    deploy: 'data-sort-deploy',
    apnic: 'data-sort-apnic',
  };

  var state = { col: null, dir: 1 };

  function parseSortNumber(val) {
    if (val === null || val === undefined || val === '') {
      return null;
    }
    var n = parseFloat(String(val));
    return isNaN(n) ? null : n;
  }

  function compareRows(a, b, col, dir) {
    var attr = sortAttr[col];
    var va = a.getAttribute(attr);
    var vb = b.getAttribute(attr);
    if (col === 'name') {
      var ca = (va || '').toLowerCase();
      var cb = (vb || '').toLowerCase();
      var cmp = ca.localeCompare(cb, undefined, { sensitivity: 'base' });
      return dir * cmp;
    }
    var na = parseSortNumber(va);
    var nb = parseSortNumber(vb);
    var aMissing = na === null;
    var bMissing = nb === null;
    if (aMissing && bMissing) {
      return 0;
    }
    if (aMissing) {
      return 1;
    }
    if (bMissing) {
      return -1;
    }
    if (na === nb) {
      return (a.getAttribute('data-sort-name') || '').localeCompare(b.getAttribute('data-sort-name') || '', undefined, { sensitivity: 'base' });
    }
    return dir * (na - nb);
  }

  function setAriaSort(activeCol) {
    var headers = table.querySelectorAll('thead th[data-sort-col]');
    for (var i = 0; i < headers.length; i++) {
      var th = headers[i];
      var c = th.getAttribute('data-sort-col');
      if (c === activeCol) {
        th.setAttribute('aria-sort', state.dir === 1 ? 'ascending' : 'descending');
      } else {
        th.setAttribute('aria-sort', 'none');
      }
    }
  }

  function sortBy(col) {
    if (state.col === col) {
      state.dir = -state.dir;
    } else {
      state.col = col;
      state.dir = col === 'name' ? 1 : -1;
    }
    var rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
    rows.sort(function (a, b) {
      return compareRows(a, b, col, state.dir);
    });
    for (var j = 0; j < rows.length; j++) {
      tbody.appendChild(rows[j]);
    }
    setAriaSort(col);
  }

  var buttons = table.querySelectorAll('thead button[data-sort-col]');
  for (var k = 0; k < buttons.length; k++) {
    (function (btn) {
      btn.addEventListener('click', function () {
        var col = btn.getAttribute('data-sort-col');
        if (col) {
          sortBy(col);
        }
      });
    })(buttons[k]);
  }

  function applyPctRampToCells() {
    var ramp = typeof window !== 'undefined' && window.PacificPctColorRamp;
    if (!ramp || typeof ramp.applyToTableCell !== 'function') {
      return;
    }
    var cells = table.querySelectorAll('td.economies-pct');
    for (var c = 0; c < cells.length; c++) {
      var td = cells[c];
      ramp.applyToTableCell(td, td.getAttribute('data-pct'));
    }
  }
  applyPctRampToCells();
})();
