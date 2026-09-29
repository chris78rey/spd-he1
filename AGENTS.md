## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Product decisions

- Before proposing or implementing a product or interface change, read `PRINCIPIOS_DE_PRODUCTO.md` and check the relevant nodes in `graphify-out/graph.json` with Graphify.
- Use its ranked real user jobs as the default decision rule: find and recover a document first, save it with confidence, recognize it before acting, organize in batches, then consider collaboration and visual polish.
- Keep these principles and their user evidence current in Graphify when product decisions change. Never advertise a simulated flow as a working feature.
