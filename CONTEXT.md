# Pool Control

The language for deciding whether Pooly or the pool owner currently governs the spa and for preparing owner-directed changes safely.

## Language

**Automatic control**:
The operating state in which schedules and reconciliation govern the spa.
_Avoid_: Automatic mode, scheduler mode

**Manual session**:
A timed or indefinite operating state in which automatic control is paused and an explicitly submitted complete pool state governs the spa.
_Avoid_: Manual mode, manual override

**Manual session draft**:
A client-local set of proposed settings and a duration that has no effect on the server or spa until it is applied.
_Avoid_: Pending override, unsaved manual mode
