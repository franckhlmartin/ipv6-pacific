/**
 * Shared 0–100% red→green ramp (same piecewise linear RGB as the EEZ map).
 */
(function (global) {
  var GRADIENT = [
    { pct: 0, hex: '#FF0000' },
    { pct: 25, hex: '#E04A24' },
    { pct: 50, hex: '#8FA822' },
    { pct: 75, hex: '#47C41B' },
    { pct: 100, hex: '#00FF00' },
  ];

  var NO_DATA_GRAY = '#9aa3b2';

  function hexToRgb(hex) {
    var h = hex.replace(/^#/, '');
    return {
      r: parseInt(h.slice(0, 2), 16),
      g: parseInt(h.slice(2, 4), 16),
      b: parseInt(h.slice(4, 6), 16),
    };
  }

  /** Interpolate along GRADIENT; pct clamped to [0, 100]. Returns rgb(...) or hex at endpoints. */
  function colorForPct(pct) {
    var p = Math.max(0, Math.min(100, pct));
    var stops = GRADIENT;
    if (p <= stops[0].pct) {
      return stops[0].hex;
    }
    if (p >= stops[stops.length - 1].pct) {
      return stops[stops.length - 1].hex;
    }
    var i = 0;
    while (i < stops.length - 1 && p > stops[i + 1].pct) {
      i++;
    }
    var a = stops[i];
    var b = stops[i + 1];
    var u = (p - a.pct) / (b.pct - a.pct);
    var ca = hexToRgb(a.hex);
    var cb = hexToRgb(b.hex);
    var r = Math.round(ca.r + (cb.r - ca.r) * u);
    var g = Math.round(ca.g + (cb.g - ca.g) * u);
    var bl = Math.round(ca.b + (cb.b - ca.b) * u);
    return 'rgb(' + r + ',' + g + ',' + bl + ')';
  }

  function rgbFromCss(bg) {
    if (!bg || typeof bg !== 'string') {
      return null;
    }
    bg = bg.trim();
    if (bg.charAt(0) === '#') {
      if (bg.length === 7) {
        return hexToRgb(bg);
      }
      return null;
    }
    var m = bg.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
    if (m) {
      return { r: +m[1], g: +m[2], b: +m[3] };
    }
    return null;
  }

  /** Pick readable text color on top of a ramp fill (hex or rgb(...)). */
  function textColorForBackground(bgCss) {
    var rgb = rgbFromCss(bgCss);
    if (!rgb) {
      return '#111827';
    }
    var L = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
    return L > 0.58 ? '#111827' : '#ffffff';
  }

  /** Style a table cell: missing/empty dataPct uses NO_DATA_GRAY; otherwise ramp fill. */
  function applyToTableCell(td, dataPctAttr) {
    if (dataPctAttr == null || String(dataPctAttr).trim() === '') {
      td.style.backgroundColor = NO_DATA_GRAY;
      td.style.color = '#374151';
      return;
    }
    var p = parseFloat(String(dataPctAttr));
    if (isNaN(p)) {
      td.style.backgroundColor = NO_DATA_GRAY;
      td.style.color = '#374151';
      return;
    }
    var bg = colorForPct(p);
    td.style.backgroundColor = bg;
    td.style.color = textColorForBackground(bg);
  }

  global.PacificPctColorRamp = {
    colorForPct: colorForPct,
    NO_DATA_GRAY: NO_DATA_GRAY,
    applyToTableCell: applyToTableCell,
    textColorForBackground: textColorForBackground,
  };
})(typeof window !== 'undefined' ? window : globalThis);
