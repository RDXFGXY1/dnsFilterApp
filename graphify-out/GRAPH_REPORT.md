# Graph Report - .  (2026-08-23)

## Corpus Check
- 63 files · ~62,453 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 864 nodes · 1698 edges · 48 communities (34 shown, 14 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 34 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Discipline.Js
- Engine
- Model
- Server
- Server
- Manager
- Db
- Server
- Dns Cli.Sh
- Aiblocker
- Server
- Engine
- Blockpageserver
- Upstreampool
- Keywordmanager
- Install.Sh
- Contentinspector
- Uninstall.Sh
- Config
- Tlsmanager
- Dashboard.Js
- Main()
- Start Dns Filter.Ps1
- Update.Sh
- Dns Content Filter
- Filtering Pipeline
- Blocklist Management
- Release Changelog
- Start.Sh
- Discipline Workspace
- Dns Filter Tui
- Dns Filter Architecture
- Installation Guide
- Cross Platform Runtime Guide
- Reinstall.Sh
- Dns Filter Management Dashboard
- Login Interface
- Deploy Website
- Go Build And Test
- Dns Filter Tui
- Github.Com/Rdxfgxy1/Dns Filter App
- Invalid Login Fix
- Background Service Scripts

## God Nodes (most connected - your core abstractions)
1. `Engine` - 67 edges
2. `api()` - 45 edges
3. `Server` - 42 edges
4. `DB` - 38 edges
5. `Model` - 37 edges
6. `Server` - 36 edges
7. `Manager` - 36 edges
8. `Server` - 31 edges
9. `Server` - 26 edges
10. `BlockPageServer` - 26 edges

## Surprising Connections (you probably didn't know these)
- `DNS Content Filter` --semantically_similar_to--> `DNS Content Filter`  [INFERRED] [semantically similar]
  PROJECT_SUMMARY.md → README.md
- `main()` --calls--> `Load()`  [INFERRED]
  cmd/server/main.go → internal/config/config.go
- `runDNSCommand()` --calls--> `ResetDNS()`  [INFERRED]
  cmd/server/main.go → internal/netdns/netdns.go
- `runDNSCommand()` --calls--> `SetDNS()`  [INFERRED]
  cmd/server/main.go → internal/netdns/netdns.go
- `main()` --calls--> `SetLevel()`  [INFERRED]
  cmd/server/main.go → pkg/logger/logger.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **DNS Filter Documentation Set** — docs_installation_installation_guide, docs_quickstart_quick_start_guide, docs_running_cross_platform_runtime_guide, docs_usage_usage_guide, docs_architecture_dns_filter_architecture [EXTRACTED 1.00]
- **Discipline Productivity System** — web_templates_discipline_discipline_workspace, web_templates_discipline_goal_project_task_tracking, web_templates_discipline_habit_and_activity_tracking [EXTRACTED 1.00]

## Communities (48 total, 14 thin omitted)

### Community 0 - "Discipline.Js"
Cohesion: 0.05
Nodes (104): addRow(), addSubtask(), addTodayTask(), addWsTask(), _allProjects, api(), calMonth(), calShowDay() (+96 more)

### Community 1 - "Engine"
Cohesion: 0.05
Nodes (18): Bundle, BlockPageServer, Category, CustomBlocklist, CustomBlocklistEntry, Engine, GuardEvent, ByID() (+10 more)

### Community 2 - "Model"
Cohesion: 0.08
Nodes (26): envOrDefault(), Client, Msg, Time, main(), NewModel(), truncate(), uptimeText() (+18 more)

### Community 3 - "Server"
Cohesion: 0.07
Nodes (8): Server, Context, RouterGroup, queryLimit(), activeInterface(), ResetDNS(), SetDNS(), Status()

### Community 4 - "Server"
Cohesion: 0.07
Nodes (23): cacheEntry, DNSCache, QueryEvent, Server, Statistics, Duration, Msg, RWMutex (+15 more)

### Community 5 - "Manager"
Cohesion: 0.08
Nodes (15): DayCount, Event, Goal, GoalPatch, Habit, Manager, Project, Reminder (+7 more)

### Community 6 - "Db"
Cohesion: 0.07
Nodes (9): Category, CategoryManager, BlockedQuery, DB, TimelinePoint, RWMutex, NewCategoryManager(), Time (+1 more)

### Community 7 - "Server"
Cohesion: 0.11
Nodes (6): HandlerFunc, Server, Context, Engine, NewServer(), verifyPassword()

### Community 8 - "Dns Cli.Sh"
Cohesion: 0.17
Nodes (33): api_del(), api_post(), BLUE, cmd_block(), cmd_clear(), cmd_disable(), cmd_enable(), cmd_help() (+25 more)

### Community 9 - "Aiblocker"
Cohesion: 0.11
Nodes (9): AIBlocker, DomainFeatures, NaiveBayesModel, PredictionResult, boolToFloat(), RWMutex, NewAIBlocker(), NewNaiveBayesModel() (+1 more)

### Community 10 - "Server"
Cohesion: 0.12
Nodes (5): discPriorityXP(), Server, Context, RouterGroup, queryInt64()

### Community 11 - "Engine"
Cohesion: 0.12
Nodes (9): Achievement, Challenge, Engine, LeaderboardEntry, Reward, UserProfile, RWMutex, Time (+1 more)

### Community 12 - "Blockpageserver"
Cohesion: 0.11
Nodes (10): BlockPageData, BlockStats, Handler, BlockPageServer, ResponseWriter, RWMutex, NewBlockPageServer(), Request (+2 more)

### Community 13 - "Upstreampool"
Cohesion: 0.12
Nodes (20): UpstreamPool, RWMutex, NewUpstreamPool(), currentDNS(), resetDNS(), run(), serviceForInterface(), setDNS() (+12 more)

### Community 14 - "Keywordmanager"
Cohesion: 0.12
Nodes (4): RWMutex, NewKeywordManager(), KeywordList, KeywordManager

### Community 15 - "Install.Sh"
Cohesion: 0.26
Nodes (22): banner(), check_binary(), check_dependencies(), check_os(), check_port_53(), check_root(), configure_dns(), configure_dns_networkmanager() (+14 more)

### Community 16 - "Contentinspector"
Cohesion: 0.14
Nodes (7): CacheEntry, ContentInspector, InspectionResult, Client, RWMutex, Time, NewContentInspector()

### Community 17 - "Uninstall.Sh"
Cohesion: 0.29
Nodes (21): banner(), check_root(), disable_service(), err(), info(), main(), ok(), remove_cli() (+13 more)

### Community 18 - "Config"
Cohesion: 0.21
Nodes (14): AdvancedConfig, BlocklistsConfig, BlocklistSource, Config, DatabaseConfig, FilteringConfig, LoggingConfig, ScheduleConfig (+6 more)

### Community 19 - "Tlsmanager"
Cohesion: 0.21
Nodes (7): tlsManager, Certificate, ClientHelloInfo, BlockPageServer, RWMutex, newTLSManager(), PrivateKey

### Community 20 - "Dashboard.Js"
Cohesion: 0.29
Nodes (13): addToWhitelist(), calculateBlockRate(), clearCache(), escapeHtml(), exportLogs(), formatTime(), loadRecentQueries(), loadStats() (+5 more)

### Community 21 - "Main()"
Cohesion: 0.29
Nodes (11): chdirToExeDir(), main(), printBanner(), runDNSCommand(), setBootstrapDNS(), Logger, Get(), init() (+3 more)

### Community 22 - "Start Dns Filter.Ps1"
Cohesion: 0.38
Nodes (3): Setup-SystemDNS(), Start-DnsFilter(), Write-Log()

### Community 25 - "Update.Sh"
Cohesion: 0.60
Nodes (3): backup_configs(), run_build_steps(), update.sh script

### Community 26 - "Dns Content Filter"
Cohesion: 0.50
Nodes (4): DNS Content Filter, Network-wide Filtering, DNS Content Filter, System DNS Setup

### Community 27 - "Filtering Pipeline"
Cohesion: 0.67
Nodes (3): DNS Filter Configuration, Filtering Pipeline, Schedule-based Filtering

### Community 28 - "Blocklist Management"
Cohesion: 0.67
Nodes (3): Custom Blocklist, Blocklist Management, Usage Guide

### Community 29 - "Release Changelog"
Cohesion: 0.67
Nodes (3): Release Changelog, Unreleased Changes, v1.0.0 Initial Baseline

### Community 33 - "Discipline Workspace"
Cohesion: 0.67
Nodes (3): Discipline Workspace, Goal Project and Task Tracking, Habit and Activity Tracking

## Knowledge Gaps
- **59 isolated node(s):** `dns-filter-tui`, `authMsg`, `GREEN`, `RED`, `YELLOW` (+54 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **14 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `DB` connect `Db` to `Engine`, `Server`, `Manager`, `Server`, `Aiblocker`, `Engine`, `Blockpageserver`, `Keywordmanager`, `Contentinspector`?**
  _High betweenness centrality (0.203) - this node is a cross-community bridge._
- **Why does `Engine` connect `Engine` to `Db`, `Aiblocker`, `Engine`, `Keywordmanager`, `Config`, `Main()`?**
  _High betweenness centrality (0.099) - this node is a cross-community bridge._
- **Why does `Server` connect `Server` to `Config`, `Main()`, `Upstreampool`, `Db`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **What connects `dns-filter-tui`, `authMsg`, `GREEN` to the rest of the system?**
  _59 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Discipline.Js` be split into smaller, more focused modules?**
  _Cohesion score 0.05366300366300366 - nodes in this community are weakly interconnected._
- **Should `Engine` be split into smaller, more focused modules?**
  _Cohesion score 0.05115089514066496 - nodes in this community are weakly interconnected._
- **Should `Model` be split into smaller, more focused modules?**
  _Cohesion score 0.08350168350168351 - nodes in this community are weakly interconnected._