(function () {
  var global = typeof window !== 'undefined' ? window : globalThis;
  var root = document.getElementById('eez-map-root');
  if (!root) {
    return;
  }

  // Labels from <title> inside each EEZ path in EEZ_Oceania.svg → ISO 3166-1 alpha-2 (monitored economies).
  var TITLE_TO_ISO = {
    'American Samoa (US)': 'AS',
    'Cook Islands (NZ)': 'CK',
    'Federated States of Micronesia': 'FM',
    'Fiji': 'FJ',
    'French Polynesia (Fr)': 'PF',
    'Kiribati (Gilbert Islands)': 'KI',
    'Line Islands (Kiribati)': 'KI',
    Marshalls: 'MH',
    Nauru: 'NR',
    'New Caledonia': 'NC',
    'Niue (NZ)': 'NU',
    'Northern Marianas (US)': 'MP',
    'Guam (US)': 'GU',
    'Papua New Guinea': 'PG',
    Palau: 'PW',
    'Phoenix Islands (Kiribati)': 'KI',
    Samoa: 'WS',
    'Solomon Islands': 'SB',
    'Tokelau (NZ)': 'TK',
    Tonga: 'TO',
    Tuvalu: 'TV',
    Vanuatu: 'VU',
    'Wallis and Futuna (Fr)': 'WF',
  };

  function buildPreferredByISO(indexPayload) {
    var out = {};
    if (!indexPayload || !indexPayload.countries) {
      return out;
    }
    for (var k = 0; k < indexPayload.countries.length; k++) {
      var row = indexPayload.countries[k];
      if (!row || !row.iso2) {
        continue;
      }
      var al = row.apnic_labs;
      if (al && typeof al.preferred_pc_raw === 'number' && !isNaN(al.preferred_pc_raw)) {
        out[String(row.iso2).toUpperCase()] = al.preferred_pc_raw;
      }
    }
    return out;
  }

  var ramp = global.PacificPctColorRamp;
  if (!ramp || typeof ramp.colorForPct !== 'function') {
    ramp = {
      colorForPct: function () {
        return '#9aa3b2';
      },
      NO_DATA_GRAY: '#9aa3b2',
    };
    console.error('PacificPctColorRamp missing; load pct-color-ramp.js before map-home.js');
  }

  Promise.all([
    fetch('/static/img/EEZ_Oceania.svg').then(function (r) {
      if (!r.ok) {
        throw new Error('fetch failed');
      }
      return r.text();
    }),
    fetch('/api/index.json')
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .catch(function () {
        return null;
      }),
  ])
    .then(function (results) {
      var svgText = results[0];
      var preferredByISO = buildPreferredByISO(results[1]);
      var parser = new DOMParser();
      var doc = parser.parseFromString(svgText, 'image/svg+xml');
      var svg = doc.documentElement;
      if (!svg || svg.querySelector('parsererror')) {
        throw new Error('invalid svg');
      }
      svg.setAttribute('class', 'eez-map-svg');
      // Intrinsic doc size from Inkscape; enables uniform scaling in CSS (no stretch).
      svg.setAttribute('viewBox', '0 0 385 215');
      svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
      svg.removeAttribute('width');
      svg.removeAttribute('height');
      svg.setAttribute('role', 'img');

      // Ocean background: source rect used oversized coords and rendered after stray paths.
      // Snap to viewBox and paint first (after defs) so it fills the visible map.
      var defs = svg.querySelector('defs');
      var ocean = svg.querySelector('#rect5538-5');
      if (ocean && defs && defs.parentNode === svg) {
        ocean.setAttribute('x', '0');
        ocean.setAttribute('y', '0');
        ocean.setAttribute('width', '385');
        ocean.setAttribute('height', '215');
        defs.parentNode.insertBefore(ocean, defs.nextSibling);
      }

      root.appendChild(svg);

      var svgNS = 'http://www.w3.org/2000/svg';
      var paths = svg.querySelectorAll('path');

      for (var j = 0; j < paths.length; j++) {
        var p = paths[j];
        var tEl = p.querySelector('title');
        if (!tEl) {
          continue;
        }
        var territoryName = tEl.textContent.replace(/\s+/g, ' ').trim();
        if (!TITLE_TO_ISO[territoryName]) {
          p.classList.add('eez-region--outside');
          p.style.setProperty('fill', '#b8bcc4');
          p.style.setProperty('stroke', '#9ca3af');
          p.style.setProperty('stroke-width', '0.25');
          p.style.setProperty('cursor', 'default');
        }
      }

      var labelLayer = document.createElementNS(svgNS, 'g');
      labelLayer.setAttribute('class', 'eez-iso-labels');
      labelLayer.setAttribute('pointer-events', 'none');

      for (var i = 0; i < paths.length; i++) {
        var path = paths[i];
        var titleEl = path.querySelector('title');
        if (!titleEl) {
          continue;
        }
        var label = titleEl.textContent.replace(/\s+/g, ' ').trim();
        var iso = TITLE_TO_ISO[label];
        if (!iso) {
          continue;
        }
        path.classList.add('eez-region--linked');
        path.setAttribute('data-iso2', iso);
        var pct = preferredByISO[iso];
        if (pct != null) {
          path.style.setProperty('fill', ramp.colorForPct(pct));
          path.setAttribute('data-ipv6-preferred-pct', String(pct));
          titleEl.textContent =
            label + ' — ' + pct.toFixed(2) + '% IPv6 preferred (APNIC Labs estimate)';
          path.setAttribute(
            'aria-label',
            label + ' — ' + pct.toFixed(2) + '% IPv6 preferred — open monitoring page'
          );
        } else {
          path.style.setProperty('fill', ramp.NO_DATA_GRAY);
          path.setAttribute('aria-label', label + ' — open monitoring page');
        }
        path.style.setProperty('stroke', '#4b5563');
        path.style.setProperty('stroke-width', '0.25');
        path.style.cursor = 'pointer';
        path.setAttribute('tabindex', '0');
        path.setAttribute('role', 'link');

        path.addEventListener('click', function (iso2) {
          return function (e) {
            e.preventDefault();
            window.location.href = '/country/' + iso2;
          };
        }(iso));

        path.addEventListener('keydown', function (iso2) {
          return function (e) {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              window.location.href = '/country/' + iso2;
            }
          };
        }(iso));

        try {
          var box = path.getBBox();
          if (box.width < 0.5 || box.height < 0.5) {
            continue;
          }
          var cx = box.x + box.width / 2;
          var cy = box.y + box.height / 2;
          var dim = Math.min(box.width, box.height);
          var fontSize = Math.min(13, Math.max(5, dim * 0.38));

          var text = document.createElementNS(svgNS, 'text');
          text.setAttribute('x', String(cx));
          text.setAttribute('y', String(cy));
          text.setAttribute('text-anchor', 'middle');
          text.setAttribute('dominant-baseline', 'central');
          text.setAttribute('font-size', String(fontSize));
          text.setAttribute('class', 'eez-iso-label');
          text.textContent = iso;
          labelLayer.appendChild(text);
        } catch (e) {
          /* ignore bbox errors */
        }
      }

      svg.appendChild(labelLayer);
    })
    .catch(function () {
      root.textContent = 'Could not load EEZ map.';
    });
})();
