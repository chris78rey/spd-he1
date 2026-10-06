---
type: "query"
date: "2026-10-02T14:29:16.157928+00:00"
question: "Why couldn't the user delete saved period WORK-AMBULATORIO-202609?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["deleteWorkspaceDialog", "periodWorkspaceID()", "deleteWorkspace()"]
---

# Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?

## Answer

Expanded from graph vocabulary: [workspace, delete, period, expediente, oracle, job]. The app only exposed whole-workspace deletion after opening a period and required manually entering an unpasteable random 8-digit confirmation code. Added a delete action beside the saved-period selector and changed confirmation to entering/pasting the exact workspace ID. The backend checks the ID and retains its existing transactional cleanup of Oracle document states and PDI_PATH before deleting local data.

## Outcome

- Signal: useful

## Source Nodes

- deleteWorkspaceDialog
- periodWorkspaceID()
- deleteWorkspace()