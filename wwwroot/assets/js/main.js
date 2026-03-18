// ── NAV ACTIVE STATE ──
(function() {
  const page = location.pathname.split('/').pop() || 'index.html';
  document.querySelectorAll('.nav-link').forEach(link => {
    const href = link.getAttribute('href');
    if (href === page || (page === '' && href === 'index.html')) {
      link.classList.add('active');
    }
  });
})();

// ── COPY BUTTONS ──
document.addEventListener('click', function(e) {
  const btn = e.target.closest('.copy-btn');
  if (!btn) return;
  const box = btn.closest('.code-box');
  if (!box) return;
  const body = box.querySelector('.code-body');
  if (!body) return;
  navigator.clipboard.writeText(body.innerText.trim()).then(() => {
    btn.textContent = 'Copied!';
    btn.classList.add('ok');
    setTimeout(() => { btn.textContent = 'Copy'; btn.classList.remove('ok'); }, 1600);
  });
});

// ── TAB SWITCHER ──
// Usage: data-tab-group="name" on container, data-tab="id" on buttons, data-tab-content="id" on panels
document.addEventListener('click', function(e) {
  const btn = e.target.closest('[data-tab]');
  if (!btn) return;
  const group = btn.closest('[data-tab-group]');
  if (!group) return;
  const id = btn.dataset.tab;
  group.querySelectorAll('[data-tab]').forEach(b => b.classList.remove('active'));
  group.querySelectorAll('[data-tab-content]').forEach(p => p.classList.remove('active'));
  btn.classList.add('active');
  const panel = group.querySelector('[data-tab-content="' + id + '"]');
  if (panel) panel.classList.add('active');
});

// ── SIDEBAR NAV (install / api) ──
document.addEventListener('click', function(e) {
  const item = e.target.closest('[data-section]');
  if (!item) return;
  const sidebar = item.closest('[data-sidebar]');
  if (!sidebar) return;
  const id = item.dataset.section;
  sidebar.querySelectorAll('[data-section]').forEach(i => i.classList.remove('active'));
  item.classList.add('active');
  const container = document.querySelector('[data-sections]');
  if (container) {
    container.querySelectorAll('[data-sec]').forEach(s => s.style.display = 'none');
    const target = container.querySelector('[data-sec="' + id + '"]');
    if (target) target.style.display = 'block';
  }
});

// ── INTERSECTION OBSERVER for scroll reveals ──
const observer = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
      observer.unobserve(entry.target);
    }
  });
}, { threshold: 0.08 });

document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── SCROLL PROGRESS BAR ──
const bar = document.getElementById('scroll-bar');
if (bar) {
  window.addEventListener('scroll', () => {
    const pct = window.scrollY / (document.body.scrollHeight - window.innerHeight) * 100;
    bar.style.width = Math.min(pct, 100) + '%';
  });
}
