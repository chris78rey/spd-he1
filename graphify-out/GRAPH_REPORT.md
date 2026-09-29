# Graph Report - documentos  (2026-09-29)

## Corpus Check
- 11 files · ~5,032 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 3 file(s) not represented in the graph (top: (none) 2, .css 1)

## Summary
- 160 nodes · 201 edges · 13 communities (10 shown, 3 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 4 edges (avg confidence: 0.62)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- App.vue
- package.json
- Find and recover documents
- scripts
- main.go
- upload
- AGENTS.md
- server
- Folio
- loadDocuments
- folio
- loadPlanilla
- install-service.sh

## God Nodes (most connected - your core abstractions)
1. `server` - 11 edges
2. `writeError()` - 7 edges
3. `writeJSON()` - 6 edges
4. `main()` - 5 edges
5. `scripts` - 5 edges
6. `Find and recover documents` - 5 edges
7. `Primary user outcome` - 5 edges
8. `session` - 4 edges
9. `oracleHasRole()` - 4 edges
10. `securityHeaders()` - 3 edges

## Surprising Connections (you probably didn't know these)
- None detected - all connections are within the same source files.

## Import Cycles
- None detected.

## Communities (13 total, 3 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.04
Nodes (39): activeDoc, activeFolder, activePage, authError, authStatus, createSpaceDialog, currentUser, customSpaces (+31 more)

### Community 1 - "package.json"
Cohesion: 0.10
Nodes (20): dependencies, idb, @mdi/font, vue, vuetify, devDependencies, vite, @vitejs/plugin-vue (+12 more)

### Community 2 - "Find and recover documents"
Cohesion: 0.24
Nodes (12): Accessible responsive use, Batch organization, Complete action feedback, Recover from mistakes, Find and recover documents, Folio product, Measure and validate, Primary user outcome (+4 more)

### Community 3 - "scripts"
Cohesion: 0.40
Nodes (5): scripts, build, dev, dev:api, preview

### Community 4 - "main.go"
Cohesion: 0.07
Nodes (27): envOr(), main(), newSessionID(), oracleCredentialError(), oracleTableName(), securityHeaders(), go_pkg_context, go_pkg_crypto_rand (+19 more)

### Community 5 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 7 - "server"
Cohesion: 0.27
Nodes (11): oracleHasRole(), writeError(), writeJSON(), context.Context, database/sql.DB, net/http.Request, net/http.ResponseWriter, sync.Mutex (+3 more)

### Community 8 - "Folio"
Cohesion: 0.50
Nodes (3): Desarrollo local, Folio, Inicio automático en Linux

### Community 9 - "loadDocuments"
Cohesion: 0.67
Nodes (3): checkSession(), loadDocuments(), signIn()

### Community 11 - "loadPlanilla"
Cohesion: 0.67
Nodes (3): loadPlanilla(), openPlanilla(), signOut()

## Knowledge Gaps
- **64 isolated node(s):** `loginRequest`, `folio`, `name`, `version`, `private` (+59 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 105 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **Why does `idb` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Why does `scripts` connect `scripts` to `package.json`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **What connects `loginRequest`, `folio`, `name` to the rest of the system?**
  _64 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.03636363636363636 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.09881422924901186 - nodes in this community are weakly interconnected._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07389162561576355 - nodes in this community are weakly interconnected._