---
# beans-oyic
title: beans-serve code quality improvements
status: todo
type: epic
priority: normal
created_at: 2026-03-13T17:48:53Z
updated_at: 2026-03-13T18:54:26Z
order: V
---

Improvements identified during comprehensive code review of both backend and frontend.


## Identified Improvements

- Avoid disk I/O while holding write lock in Core.Update() (beans-rren)
- Replace findMainBeanFile directory scan with in-memory lookup (beans-fydu)
- Build incoming link index to replace O(n) FindIncomingLinks scan (beans-omee)
- Consolidate duplicate ETag error types (beans-541b)
- Improve subscription backpressure handling (beans-r7oc)
- Improve test coverage for agent manager and GraphQL subscriptions (beans-7u0s)
- Fix suppressed a11y warnings in frontend components (beans-4mua)
- Extract duplicate GraphQL mutation definitions to shared modules (beans-1edn)
- Fix navigator.platform access at module scope in FilterInput (beans-p2pw)
