(function () {
  function applyPctStyle(el) {
    if (!window.PacificPctColorRamp) {
      return;
    }
    var raw = el.getAttribute('data-pct');
    if (raw == null || String(raw).trim() === '') {
      el.style.backgroundColor = window.PacificPctColorRamp.NO_DATA_GRAY;
      el.style.color = '#374151';
      return;
    }
    var p = parseFloat(String(raw));
    if (isNaN(p)) {
      el.style.backgroundColor = window.PacificPctColorRamp.NO_DATA_GRAY;
      el.style.color = '#374151';
      return;
    }
    var bg = window.PacificPctColorRamp.colorForPct(p);
    el.style.backgroundColor = bg;
    el.style.color = window.PacificPctColorRamp.textColorForBackground(bg);
  }

  var nodes = document.querySelectorAll('.summary-strip .stat-value--pct');
  for (var i = 0; i < nodes.length; i++) {
    applyPctStyle(nodes[i]);
  }
})();
