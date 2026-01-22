# Performance Hardening Playbook

Checklist
- [ ] Identify hot paths and add benchmarks.
- [ ] Measure p95/p99 latency; set baselines.
- [ ] Remove N+1 queries and excessive allocations.
- [ ] Add caching where safe (document invalidation).
- [ ] Add profiling notes (cpu/mem) to outbox report.
