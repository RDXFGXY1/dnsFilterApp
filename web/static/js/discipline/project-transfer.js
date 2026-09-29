let validatedProjectYAML = '';
const PROJECT_YAML_TEMPLATE = `version: 1
projects:
  - name: Launch My Website
    description: Build and release a useful personal website.
    objective: A fast, accessible website is live and receives its first users.
    status: planning
    color: "#d1553b"
    start_date: 2026-09-01
    target_date: 2026-10-15
    notes: Keep the first release focused on the essential experience.
    links:
      - https://example.com/design

    milestones:
      - name: Research complete
        status: planned
        target_date: 2026-09-07
      - name: MVP ready
        status: planned
        target_date: 2026-10-01

    cycles:
      - name: Week 1
        status: planned
        start_date: 2026-09-01
        end_date: 2026-09-07

    tasks:
      - title: Research target users
        status: todo
        priority: high
        size: M
        energy: high
        milestone: Research complete
        cycle: Week 1
        estimate_minutes: 180
        labels: [research, product]
        checklist:
          - text: Prepare interview questions
          - text: Interview five users
          - text: Summarize patterns
        subtasks:
          - title: Recruit interview participants
            priority: medium

      - title: Build the homepage
        status: todo
        priority: high
        milestone: MVP ready
        estimate_minutes: 360
        depends_on: [research-users]
        checklist:
          - text: Implement responsive layout
          - text: Test keyboard navigation

      - title: Explore advanced analytics
        priority: low
        backlog: true
`;

function openProjectYAMLImport(){document.querySelector('#project-yaml-modal h2').textContent='Import Project YAML';document.querySelector('#project-yaml-modal .panel-desc').textContent='Upload or paste a complete project plan. Validation and preview happen before any project is changed.';document.getElementById('project-yaml-modal').hidden=false;document.body.classList.add('project-transfer-open')}
function closeProjectYAMLImport(){document.getElementById('project-yaml-modal').hidden=true;document.body.classList.remove('project-transfer-open')}
async function loadProjectYAMLFile(input){const file=input.files?.[0];if(!file)return;if(!/\.ya?ml$/i.test(file.name)){notify('Choose a .yaml or .yml file',true);return}document.getElementById('project-yaml-input').value=await file.text();validatedProjectYAML='';document.getElementById('project-yaml-confirm').disabled=true;previewProjectYAML()}
function loadProjectYAMLTemplate(){document.getElementById('project-yaml-input').value=PROJECT_YAML_TEMPLATE;validatedProjectYAML='';document.getElementById('project-yaml-confirm').disabled=true}
async function copyProjectYAMLPrompt(){const prompt=`Create a realistic project plan using exactly this YAML schema. Keep version 1. Do not add task keys: the app generates portable keys automatically from task titles. Valid project statuses: planning, active, paused, at_risk, completed. Valid task statuses: todo, in_progress, done. Valid priorities: low, medium, high. For depends_on, use the lowercase hyphenated title of the earlier task (for example Research target users becomes research-target-users). Use milestone and cycle names exactly as declared. Return YAML only, without markdown fences.\n\n${PROJECT_YAML_TEMPLATE}`;try{await navigator.clipboard.writeText(prompt);notify('AI project prompt copied')}catch{notify('Clipboard access unavailable',true)}}

async function previewProjectYAML(){const source=document.getElementById('project-yaml-input').value,out=document.getElementById('project-yaml-preview'),button=document.getElementById('project-yaml-confirm');validatedProjectYAML='';button.disabled=true;out.innerHTML='<span class="no-data">Validating the complete project graph…</span>';const r=await api('POST','/projects/validate-yaml',{yaml:source});if(!r||r.error){out.innerHTML=`<div class="project-yaml-error"><strong>Cannot import yet</strong><span>${esc(r?.error||'Validation failed')}</span></div>`;return}validatedProjectYAML=source;button.disabled=false;out.innerHTML=`<div class="project-preview-summary"><div><span>Projects</span><strong>${r.projects}</strong></div><div><span>Milestones</span><strong>${r.milestones}</strong></div><div><span>Tasks + subtasks</span><strong>${r.tasks}</strong></div><div><span>Validation</span><strong>Ready</strong></div></div><div class="project-preview-list">${r.document.projects.map(renderProjectYAMLPreview).join('')}</div>`}
function renderProjectYAMLPreview(project){const depCount=project.tasks.reduce((n,t)=>n+(t.depends_on?.length||0)+(t.subtasks||[]).reduce((a,s)=>a+(s.depends_on?.length||0),0),0),checkCount=project.tasks.reduce((n,t)=>n+(t.checklist?.length||0)+(t.subtasks||[]).reduce((a,s)=>a+(s.checklist?.length||0),0),0),backlog=project.tasks.filter(t=>t.backlog).length;return `<article class="project-preview-card"><div class="project-preview-head"><div><span>${esc(project.status)}</span><h3>${esc(project.name)}</h3><p>${esc(project.objective||project.description||'No objective provided')}</p></div><b>${project.tasks.length} main tasks</b></div><div class="project-preview-metrics"><span>${project.milestones.length} milestones</span><span>${project.cycles.length} cycles</span><span>${depCount} dependencies</span><span>${checkCount} checklist items</span><span>${backlog} backlog ideas</span></div><div class="project-preview-tree">${project.tasks.slice(0,8).map(t=>`<div><i class="priority-${esc(t.priority)}"></i><strong>${esc(t.title)}</strong><small>${esc(t.key)}${t.milestone?' · '+esc(t.milestone):''}</small>${t.subtasks?.length?`<span>${t.subtasks.length} subtasks</span>`:''}</div>`).join('')}${project.tasks.length>8?`<em>+ ${project.tasks.length-8} more tasks</em>`:''}</div></article>`}

async function confirmProjectYAMLImport(){if(!validatedProjectYAML){notify('Validate the YAML first',true);return}const mode=document.querySelector('input[name="project-yaml-mode"]:checked')?.value||'add';if(mode==='replace'&&!confirm('Replace only projects whose names match this YAML file? Other projects will stay untouched.'))return;const r=await api('POST','/projects/import-yaml',{yaml:validatedProjectYAML,mode});if(r?.success){notify(`Imported ${r.projects} projects and ${r.tasks} tasks${r.merged?`; merged into ${r.merged} project${r.merged===1?'':'s'}`:''}${r.skipped?`; skipped ${r.skipped}`:''}`);closeProjectYAMLImport();loadProjects();if(currentProject)loadWorkspace()}else notify(r?.error||'Project import failed',true)}
function exportCurrentProjectYAML(event){event.preventDefault();if(!currentProject){notify('Open a project first',true);return}window.location.href='/api/discipline/projects/export-yaml?project='+encodeURIComponent(currentProject)}

async function openProjectExportPicker(){const modal=document.getElementById('project-export-modal'),list=document.getElementById('project-export-list');modal.hidden=false;document.body.classList.add('project-transfer-open');list.innerHTML='<span class="no-data">Loading projects…</span>';const projects=await api('GET','/projects');if(!projects){list.innerHTML='<span class="no-data">Projects could not be loaded.</span>';return}list.innerHTML=projects.length?projects.map(p=>`<label class="project-export-option"><input type="checkbox" value="${p.id}" checked onchange="updateProjectExportCount()"><span><strong>${esc(p.name)}</strong><span>${esc(p.objective||p.description||'No objective')}</span></span><b>${p.completed_tasks||0}/${p.total_tasks||0} tasks</b></label>`).join(''):'<span class="no-data">There are no projects to export.</span>';updateProjectExportCount()}
function closeProjectExportPicker(){document.getElementById('project-export-modal').hidden=true;document.body.classList.remove('project-transfer-open')}
function setAllProjectExports(checked){document.querySelectorAll('#project-export-list input[type="checkbox"]').forEach(input=>input.checked=checked);updateProjectExportCount()}
function updateProjectExportCount(){const checked=document.querySelectorAll('#project-export-list input:checked').length;document.getElementById('project-export-count').textContent=`${checked} selected`}
function downloadSelectedProjectsYAML(){const ids=[...document.querySelectorAll('#project-export-list input:checked')].map(input=>input.value);if(!ids.length){notify('Select at least one project',true);return}window.location.href='/api/discipline/projects/export-yaml?projects='+encodeURIComponent(ids.join(','));closeProjectExportPicker()}

function yamlQuoted(value){return JSON.stringify(String(value||'Project'))}
function openTaskYAMLImport(){if(!currentProject){notify('Open a project first',true);return}openProjectYAMLImport();document.querySelector('#project-yaml-modal h2').textContent='Import Tasks into This Project';document.querySelector('#project-yaml-modal .panel-desc').textContent='Paste tasks and subtasks for this project. Existing work stays in place and new task identities are generated automatically.';document.querySelector('input[name="project-yaml-mode"][value="merge"]').checked=true;const projectName=document.getElementById('ws-name')?.textContent?.trim()||'Current Project';document.getElementById('project-yaml-input').value=`version: 1\nprojects:\n  - name: ${yamlQuoted(projectName)}\n    tasks:\n      - title: Research the next step\n        priority: high\n        checklist:\n          - text: Collect useful references\n          - text: Record the decision\n        subtasks:\n          - title: Summarize findings\n\n      - title: Build the first version\n        priority: medium\n        depends_on: [research-the-next-step]\n`;validatedProjectYAML='';document.getElementById('project-yaml-confirm').disabled=true;document.getElementById('project-yaml-preview').innerHTML='<div class="dense-empty"><strong>Task template ready</strong><span>Edit it, then choose Validate & preview. Keys will be created automatically.</span></div>'}
