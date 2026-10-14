(function () {
  const icons = {
    dashboard: '<path d="M4 4h6v6H4zM14 4h6v4h-6zM14 12h6v8h-6zM4 14h6v6H4z"/>',
    monitor: '<path d="M3 12h4l2-6 4 12 2-6h6"/>',
    protection: '<path d="M12 3 19 6v5c0 4.8-2.8 8-7 10-4.2-2-7-5.2-7-10V6l7-3Z"/>',
    automation: '<path d="M12 2v3M12 19v3M4.9 4.9 7 7M17 17l2.1 2.1M2 12h3M19 12h3M4.9 19.1 7 17M17 7l2.1-2.1"/><circle cx="12" cy="12" r="4"/>',
    advanced: '<path d="M4 20h16M6 20V9l6-5 6 5v11M9 13h6M9 16h6"/>',
    system: '<circle cx="12" cy="12" r="3"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.4-2.4 1A7 7 0 0 0 15 6l-.3-2.6h-4L10.4 6A7 7 0 0 0 8 7.1l-2.4-1-2 3.4 2 1.5a7 7 0 0 0 0 2l-2 1.5 2 3.4 2.4-1A7 7 0 0 0 10.4 18l.3 2.6h4L15 18a7 7 0 0 0 1.5-1.1l2.4 1 2-3.4-2-1.5a7 7 0 0 0 .1-1Z"/>'
  };
  const groups = [
    ['dashboard', 'Dashboard', ['overview', 'security']],
    ['monitor', 'Monitor', ['live', 'networktest', 'connections', 'alerts', 'destinations', 'downloads', 'routewatch', 'activity']],
    ['protection', 'Protection', ['protection', 'privacy', 'rules']],
    ['automation', 'Automation', ['profiles', 'schedules', 'focus', 'wellbeing']],
    ['advanced', 'Advanced', ['learning', 'decisions', 'freeze']],
    ['system', 'System', ['hardsecurity', 'plugins', 'settings', 'requests']]
  ];
  const icon = path => `<svg viewBox="0 0 24 24" aria-hidden="true">${path}</svg>`;

  function organize() {
    const nav = document.querySelector('.sidenav');
    if (!nav || nav.dataset.organized) return;
    nav.dataset.organized = '1';
    const links = {};
    nav.querySelectorAll('a[data-view]').forEach(link => {
      links[link.dataset.view] = link;
      link.classList.add('sidenav-link');
      const text = [...link.childNodes].find(node => node.nodeType === Node.TEXT_NODE && node.textContent.trim());
      if (text) {
        const copy = document.createElement('span');
        copy.className = 'nav-item-copy';
        copy.textContent = text.textContent.trim();
        link.replaceChild(copy, text);
      }
    });

    nav.replaceChildren();
    const current = document.createElement('div');
    current.className = 'current-console';
    current.innerHTML = `${icon(icons.protection)}<div><small>Current workspace</small><strong>Protection Console</strong></div><i></i>`;
    nav.appendChild(current);

    const host = document.createElement('div');
    host.className = 'nav-groups';
    nav.appendChild(host);
    groups.forEach(([id, label, views]) => {
      const available = views.map(view => links[view]).filter(Boolean);
      if (!available.length) return;
      const group = document.createElement('section');
      group.className = 'nav-group';
      group.dataset.navGroup = id;
      group.innerHTML = `<div class="nav-group-title">${icon(icons[id])}<span>${label}</span></div><div class="nav-group-items"></div>`;
      const items = group.querySelector('.nav-group-items');
      available.forEach(link => items.appendChild(link));
      host.appendChild(group);
    });

    const discipline = document.createElement('a');
    discipline.className = 'discipline-launcher';
    discipline.href = '/discipline';
    discipline.innerHTML = `<span class="discipline-icon">${icon('<path d="m6 12 4 4 8-9"/><path d="M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z"/>')}</span><span><small>Personal command system</small><strong>Open Discipline</strong><em>Focus, plan and execute</em></span>${icon('<path d="m9 5 7 7-7 7"/>')}`;
    nav.appendChild(discipline);
    markActive();
    new MutationObserver(markActive).observe(nav, { subtree: true, attributes: true, attributeFilter: ['class'] });
  }

  function markActive() {
    document.querySelectorAll('.nav-group').forEach(group => group.classList.toggle('has-active', !!group.querySelector('a.active')));
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', organize);
  else organize();
})();
