(function () {
  function parseSortNumber(cell) {
    if (!cell) return 0;
    const dv = cell.getAttribute('data-sort-value');
    if (dv !== null && dv !== '') {
      const n = parseInt(dv, 10);
      if (Number.isFinite(n)) return n;
    }
    const n = parseInt(cell.textContent.trim(), 10);
    return Number.isFinite(n) ? n : 0;
  }

  function initSortableTable(table) {
    const tbody = table.querySelector('tbody');
    if (!tbody) return;

    const triggers = table.querySelectorAll('th.sortable .sort-trigger');
    if (!triggers.length) return;

    let activeCol = null;
    let ascending = true;

    function cellSortKeyText(row, colIndex) {
      const cell = row.cells[colIndex];
      if (!cell) return '';
      const link = cell.querySelector('a');
      const raw = link ? link.textContent : cell.textContent;
      return raw.trim().toLowerCase();
    }

    function setAriaSort() {
      table.querySelectorAll('thead th.sortable').forEach(function (th) {
        const btn = th.querySelector('.sort-trigger');
        if (!btn) return;
        const idx = parseInt(btn.getAttribute('data-sort-index'), 10);
        const ind = th.querySelector('.sort-indicator');
        if (idx === activeCol) {
          th.setAttribute('aria-sort', ascending ? 'ascending' : 'descending');
          if (ind) ind.textContent = ascending ? '\u25b2' : '\u25bc';
        } else {
          th.removeAttribute('aria-sort');
          if (ind) ind.textContent = '';
        }
      });
    }

    function sortRows(colIndex, asc, mode) {
      const rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
      rows.sort(function (a, b) {
        if (mode === 'number') {
          const na = parseSortNumber(a.cells[colIndex]);
          const nb = parseSortNumber(b.cells[colIndex]);
          const cmp = na - nb;
          return asc ? cmp : -cmp;
        }
        const va = cellSortKeyText(a, colIndex);
        const vb = cellSortKeyText(b, colIndex);
        const cmp = va.localeCompare(vb, undefined, { sensitivity: 'base', numeric: true });
        return asc ? cmp : -cmp;
      });
      rows.forEach(function (r) {
        tbody.appendChild(r);
      });
    }

    triggers.forEach(function (btn) {
      btn.addEventListener('click', function () {
        const idx = parseInt(btn.getAttribute('data-sort-index'), 10);
        if (Number.isNaN(idx)) return;
        const mode = btn.getAttribute('data-sort-mode') || 'text';
        if (idx === activeCol) ascending = !ascending;
        else {
          activeCol = idx;
          ascending = true;
        }
        sortRows(activeCol, ascending, mode);
        setAriaSort();
      });
    });
  }

  document.querySelectorAll('table.sortable-table').forEach(initSortableTable);
})();
