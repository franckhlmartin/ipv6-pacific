/**
 * Apply PacificPctColorRamp to DMARC and RPKI cells on country pages.
 */
(function () {
  function applyRampCells(selector) {
    if (!window.PacificPctColorRamp) {
      return;
    }
    document.querySelectorAll(selector).forEach(function (td) {
      window.PacificPctColorRamp.applyToTableCell(td, td.getAttribute('data-pct'));
    });
  }
  function init() {
    applyRampCells('.results td.col-dmarc-pct');
    applyRampCells('.bgphe-table td.col-rpki-pct');
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
