## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Runtime build freshness

- When a user asks to verify a change in the running application, identify the active service process and the executable it has loaded. Do not infer that the running application uses the current source or `bin/folio-server` from file timestamps alone.
- Build the current source to a temporary candidate and compare its SHA-256 with `/proc/<MainPID>/exe`; also compare with `bin/folio-server` when that is the service executable. If the hashes differ, state that the running instance is stale.
- Before replacing an ignored or otherwise untracked executable, back it up and confirm a rollback path. Check that no child job is active before restarting the service.
- Replacing the service executable or restarting the service is a deployment. Do it only with explicit user authorization. After an authorized restart, verify the service is active and the loaded executable hash matches the candidate; if startup fails, restore the backup and verify the previous service is active.

## Product decisions

- Before proposing or implementing a product or interface change, read `PRINCIPIOS_DE_PRODUCTO.md` and check the relevant nodes in `graphify-out/graph.json` with Graphify.
- Use its ranked real user jobs as the default decision rule: find and recover a document first, save it with confidence, recognize it before acting, organize in batches, then consider collaboration and visual polish.
- Keep these principles and their user evidence current in Graphify when product decisions change. Never advertise a simulated flow as a working feature.
