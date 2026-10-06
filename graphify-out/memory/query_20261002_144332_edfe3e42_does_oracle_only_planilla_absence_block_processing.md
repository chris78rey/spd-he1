---
type: "query"
date: "2026-10-02T14:43:32.193731+00:00"
question: "Does Oracle-only planilla absence block processing when ZIP is the source of truth?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["ingestPreviewReadyToPrepare", ".inspectIngestZIP()", ".buildPatientFolders()"]
---

# Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?

## Answer

Expanded from graph vocabulary: [zip, oracle, preview, prepare, folder]. The ready-to-prepare condition requires folders and PDFs in the ZIP, no invalid entries, and ZIP folders matched to Oracle with patient names; it does not require every Oracle planilla to appear in the ZIP. The backend processes ZIP folders and skips Oracle-only rows. Updated the UX to state this explicitly and hide the list of omitted PDI_TRAMITE values under a details disclosure.

## Outcome

- Signal: useful

## Source Nodes

- ingestPreviewReadyToPrepare
- .inspectIngestZIP()
- .buildPatientFolders()