(function () {
  var MODE_LABEL = {
    ipv4: 'IPv4 only',
    ipv6: 'IPv6 only',
    dual: 'Dual stack',
    ipv4outage: 'IPv4 maintenance today'
  };

  function $(id) {
    return document.getElementById(id);
  }

  function formatAddr(v) {
    if (v == null || v === '') return 'Not available';
    return String(v);
  }

  function formatPreferred(p) {
    if (!p || !p.ip) return 'Not available';
    var fam =
      p.family === 'ipv6' ? 'IPv6' : p.family === 'ipv4' ? 'IPv4' : '';
    if (fam) return String(p.ip) + ' (' + fam + ')';
    return String(p.ip);
  }

  function preferredButtonSuffix(preferred) {
    if (!preferred || !preferred.family) return '';
    if (preferred.family === 'ipv6') return 'IPv6 preferred';
    if (preferred.family === 'ipv4') return 'IPv4 preferred';
    return '';
  }

  function ensureButtonTextWrap(btn, labelEl) {
    var wrap = btn.querySelector('.conn-status-btn__text');
    if (!wrap) {
      wrap = document.createElement('span');
      wrap.className = 'conn-status-btn__text';
      if (!labelEl.parentNode) return null;
      labelEl.parentNode.insertBefore(wrap, labelEl);
      wrap.appendChild(labelEl);
    }
    var prefEl = wrap.querySelector('.conn-status-btn__pref');
    if (!prefEl) {
      prefEl = document.createElement('span');
      prefEl.className = 'conn-status-btn__pref';
      wrap.appendChild(prefEl);
    }
    return prefEl;
  }

  function updateButton(btn, mode, preferred) {
    var labelEl = btn.querySelector('.conn-status-btn__label');
    if (!labelEl) return;

    var mainLabel = MODE_LABEL[mode] || '—';
    var prefSuffix =
      mode === 'dual' ? preferredButtonSuffix(preferred) : '';
    labelEl.textContent = mainLabel;

    var prefEl = ensureButtonTextWrap(btn, labelEl);
    if (prefEl) {
      if (prefSuffix) {
        prefEl.textContent = prefSuffix;
        prefEl.hidden = false;
      } else {
        prefEl.textContent = '';
        prefEl.hidden = true;
      }
    }

    var ariaDetail = prefSuffix ? mainLabel + ', ' + prefSuffix : mainLabel;
    btn.setAttribute('aria-label', 'Your connection: ' + ariaDetail + '. Open details.');
    btn.removeAttribute('aria-disabled');
    btn.disabled = false;
    btn.setAttribute('aria-busy', 'false');

    btn.classList.remove(
      'conn-status-btn--ipv4',
      'conn-status-btn--ipv6',
      'conn-status-btn--dual',
      'conn-status-btn--pref-ipv4',
      'conn-status-btn--pref-ipv6'
    );
    if (mode === 'ipv4' || mode === 'ipv4outage') btn.classList.add('conn-status-btn--ipv4');
    if (mode === 'ipv6') btn.classList.add('conn-status-btn--ipv6');
    if (mode === 'dual') btn.classList.add('conn-status-btn--dual');
    if (mode === 'dual' && preferred && preferred.family === 'ipv4') {
      btn.classList.add('conn-status-btn--pref-ipv4');
    }
    if (mode === 'dual' && preferred && preferred.family === 'ipv6') {
      btn.classList.add('conn-status-btn--pref-ipv6');
    }
  }

  window.__updateConnStatus = function (state) {
    var btn = $('conn-status-btn');
    var dd4 = $('conn-status-ipv4');
    var dd6 = $('conn-status-ipv6');
    var ddPref = $('conn-status-preferred');
    if (!btn || !dd4 || !dd6) return;

    dd4.textContent = formatAddr(state.ipv4);
    dd6.textContent = formatAddr(state.ipv6);
    if (ddPref) ddPref.textContent = formatPreferred(state.preferred);

    updateButton(btn, state.mode, state.preferred);
  };

  function init() {
    var btn = $('conn-status-btn');
    var dlg = $('conn-status-dialog');
    if (!btn || !dlg) return;

    var closeBtn = dlg.querySelector('.conn-status-dialog__close');
    btn.addEventListener('click', function () {
      if (btn.disabled) return;
      dlg.showModal();
      btn.setAttribute('aria-expanded', 'true');
      if (closeBtn) closeBtn.focus();
    });
    dlg.addEventListener('close', function () {
      btn.setAttribute('aria-expanded', 'false');
      btn.focus();
    });
    if (closeBtn) {
      closeBtn.addEventListener('click', function () {
        dlg.close();
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
