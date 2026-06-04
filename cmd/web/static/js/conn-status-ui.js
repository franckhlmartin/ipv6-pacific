(function (global) {
  var ns = (global.IPv6PacificConnStatus = global.IPv6PacificConnStatus || {});

  var MODE_LABEL = {
    ipv4: 'IPv4 only',
    ipv6: 'IPv6 only',
    dual: 'Dual stack',
    ipv4outage: 'IPv4 maintenance today',
  };

  var VARIANT_COPY = {
    site: {
      title: 'Your connection to this site',
      subtitle: 'How we categorized your connection.',
      showEmbedCTA: true,
      showAttribution: false,
    },
    embed: {
      title: 'Your connection',
      subtitle: 'How we categorized your connection to the Pacific IPv6 probe network.',
      showEmbedCTA: false,
      showAttribution: false,
    },
    embedScript: {
      title: 'Your connection',
      subtitle: 'How we categorized your connection to the Pacific IPv6 probe network.',
      showEmbedCTA: true,
      showAttribution: false,
    },
    embedDetails: {
      title: 'Your connection',
      subtitle: 'How we categorized your connection to the Pacific IPv6 probe network.',
      showEmbedCTA: false,
      showAttribution: true,
    },
    outage566: {
      title: 'Your connection',
      subtitle: 'You are viewing this page over IPv4 during our monthly IPv6 drill.',
      showEmbedCTA: false,
      showAttribution: false,
    },
  };

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
    var prefSuffix = mode === 'dual' ? preferredButtonSuffix(preferred) : '';
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

  function isInIframe() {
    try {
      return global.parent && global.parent !== global;
    } catch (e) {
      return true;
    }
  }

  function openDetailsPopup(detailsURL) {
    var features =
      'width=440,height=560,scrollbars=yes,resizable=yes,noopener,noreferrer';
    var popup = global.open(detailsURL, 'ipv6-pacific-conn-details', features);
    if (!popup) {
      global.open(detailsURL, '_blank', 'noopener,noreferrer');
    }
  }

  function applyVariantCopy(root, variant) {
    var copy = VARIANT_COPY[variant] || VARIANT_COPY.site;
    var titleEl = root.querySelector('.conn-status-dialog__title');
    var subtitleEl = root.querySelector('.conn-status-dialog__subtitle');
    var embedCTA = root.querySelector('.conn-status-dialog__embed');
    var attribution = root.querySelector('.conn-status-attribution');
    if (titleEl) titleEl.textContent = copy.title;
    if (subtitleEl) subtitleEl.textContent = copy.subtitle;
    if (embedCTA) embedCTA.hidden = !copy.showEmbedCTA;
    if (attribution) attribution.hidden = !copy.showAttribution;
  }

  ns.initUI = function (rootEl, opts) {
    opts = opts || {};
    var root = rootEl || global.document;
    var variant = opts.variant || 'site';
    var detailsOnly = !!opts.detailsOnly;
    var btn = root.querySelector('.conn-status-btn');
    var dlg = root.querySelector('.conn-status-dialog');
    var modeBadge = root.querySelector('[data-conn-mode]');
    var dd4 =
      root.querySelector('[data-conn-ipv4]') || root.querySelector('#conn-status-ipv4');
    var dd6 =
      root.querySelector('[data-conn-ipv6]') || root.querySelector('#conn-status-ipv6');
    var ddPref =
      root.querySelector('[data-conn-preferred]') ||
      root.querySelector('#conn-status-preferred');

    applyVariantCopy(root, variant);

    function update(state) {
      if (dd4) dd4.textContent = formatAddr(state.ipv4);
      if (dd6) dd6.textContent = formatAddr(state.ipv6);
      if (ddPref) ddPref.textContent = formatPreferred(state.preferred);
      if (modeBadge) {
        modeBadge.textContent = MODE_LABEL[state.mode] || '—';
      }
      if (btn) updateButton(btn, state.mode, state.preferred);
    }

    if (detailsOnly) {
      return { update: update };
    }

    if (btn && dlg) {
      var closeBtn = dlg.querySelector('.conn-status-dialog__close');
      var detailsPopupURL = opts.detailsPopupURL || '/embed/conn-status/details';
      var useDetailsPopup = variant === 'embed' && isInIframe();

      btn.addEventListener('click', function () {
        if (btn.disabled) return;
        if (useDetailsPopup) {
          openDetailsPopup(detailsPopupURL);
          return;
        }
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

    return { update: update };
  };

  var siteUI = null;

  global.__updateConnStatus = function (state) {
    if (!siteUI) {
      siteUI = ns.initUI(global.document, { variant: 'site' });
    }
    siteUI.update(state);
  };

  function initSitePage() {
    var btn = global.document.getElementById('conn-status-btn');
    if (!btn) return;
    siteUI = ns.initUI(global.document, { variant: 'site' });
  }

  if (global.document) {
    if (global.document.readyState === 'loading') {
      global.document.addEventListener('DOMContentLoaded', initSitePage);
    } else {
      initSitePage();
    }
  }
})(typeof window !== 'undefined' ? window : this);
