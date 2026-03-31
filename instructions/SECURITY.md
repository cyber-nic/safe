# Safe Security Assessment

## 1. Purpose

This document is a security assessment of the current Safe product definition, system design, and implementation plan in:

- [PRODUCT.md](./instructions/PRODUCT.md)
- [SYSTEM_DESIGN.md](./instructions/SYSTEM_DESIGN.md)
- [IMPLEMENTATION_PLAN.md](./instructions/IMPLEMENTATION_PLAN.md)

It evaluates whether the proposed design can plausibly deliver a zero-knowledge secret manager for individuals and small trusted groups, identifies the major security strengths and weaknesses, and defines the controls and release gates required for a safe v1.

This is an assessment of the design and plan, not a claim that the system is secure today.

## 2. Executive Summary

The design is directionally strong. The most important architectural decisions are correct:

- client-side encryption is the confidentiality boundary
- the control plane is intentionally thin
- object storage is treated as a dumb durable store rather than a trusted data processor
- device enrollment is explicit
- recovery is designed around re-wrapping a random account master key instead of decrypting on the server
- revocation is framed as forward-looking, not retroactive

Those choices give Safe a credible path to a real zero-knowledge product.

The main risk is not the cryptographic hierarchy. The main risk is integrity and authorization around mutable metadata and distributed state:

- head pointers, snapshots, account config, memberships, and device state are small but security-critical
- object storage is not transactional and will happily preserve inconsistent or replayed state unless the protocol prevents it
- the control plane cannot read secrets, but it can still cause rollback, misrouting, over-broad authorization, or unauthorized device enrollment if its authority is not tightly constrained
- the browser extension is a high-risk plaintext exposure surface

The current design is viable for v1 if several items are treated as release blockers:

1. Authenticated mutable metadata with rollback detection must be mandatory, not optional.
2. Signed URL or scoped credential issuance must be path-scoped, short-lived, non-listing by default, and tested for replay and prefix overreach.
3. Device enrollment and invite acceptance must bind identities, devices, and key material explicitly.
4. Local decrypted derivatives, especially the search index and extension memory, must be minimized and wiped reliably.
5. Logging, telemetry, crash reporting, import/export, and debug tooling must be designed as data-exfiltration surfaces, not secondary concerns.

If those items are implemented rigorously, the proposed v1 can meet a reasonable security bar for a personal and small-group secret manager. If they are deferred or left ambiguous, the product will likely fail in exactly the places zero-knowledge systems usually fail: state integrity, capability scoping, client compromise surfaces, and operational leakage.

## 3. Security Goals

The design implies the following security goals for v1.

### 3.1 Primary Goals

- The service must not be able to decrypt vault contents in normal operation.
- Compromise of object storage must expose ciphertext and limited metadata, not plaintext secrets.
- Compromise of the control plane must not directly yield vault plaintext.
- Loss or theft of a locked device must not expose usable plaintext or reusable key material.
- Device addition must require possession of existing trust material, not just OAuth identity.
- Sharing must allow authorized future access while making revocation effective for future data.
- Clients must detect stale or replayed mutable state rather than silently accepting rollback.

### 3.2 Secondary Goals

- Metadata leakage should be minimized and explicitly documented.
- Offline use should not degrade security assumptions.
- Recovery should not rely on server escrow.
- Operational tooling should not become an accidental plaintext side channel.

### 3.3 Non-Goals That Must Remain Non-Goals

- protection from a fully compromised unlocked endpoint
- retroactive erasure of data already seen by a revoked member
- server-side recovery of user data
- server-side inspection, indexing, or analytics over secret contents

The non-goals are reasonable, but they must remain visible in product language. Zero knowledge is often oversold through omission rather than explicit falsehood.

## 4. Assets and Security-Critical Data

The design contains several distinct asset classes with different sensitivity.

### 4.1 Highest Sensitivity

- account master key (AMK)
- collection keys
- item data keys
- device private keys
- plaintext secret payloads
- recovery key
- decrypted local search index

Exposure of these assets compromises confidentiality directly.

### 4.2 High Sensitivity

- wrapped AMK and wrapped collection keys
- invites and enrollment payloads
- account config
- head pointers and snapshot pointers
- membership state
- signed URLs or scoped object credentials

Exposure does not necessarily reveal plaintext, but tampering can cause unauthorized access, rollback, or availability loss.

### 4.3 Moderate Sensitivity

- item IDs, collection IDs, device IDs
- object sizes
- timestamps
- event ordering and traffic patterns
- account-region mapping

These are metadata, but still sensitive. For a secret manager, metadata can reveal behavioral and organizational structure.

## 5. Trust Boundaries

The trust model in the system design is mostly sound. The important boundaries are:

### 5.1 Trusted Boundary

- unlocked client runtime on a healthy device
- vetted cryptographic implementations
- platform secure storage where available

This boundary is narrow and appropriate.

### 5.2 Untrusted but Authorized Boundary

- control plane
- object storage
- network

This means confidentiality cannot depend on server honesty. It also means integrity cannot depend solely on server-provided metadata.

### 5.3 Weakest Practical Boundary

- browser extension content-script and page interaction surface
- local OS/browser storage for decrypted derivatives
- logs, traces, crash dumps, analytics, and debugging output

These are the most likely places for real-world leakage even if the cryptographic model is correct.

## 6. Architectural Strengths

The current design has several meaningful strengths.

### 6.1 Correct Separation of Identity and Cryptographic Authority

OAuth establishes account identity, while the master password and recovery key establish cryptographic access. This is the right split. It prevents a control-plane-only compromise from becoming an automatic plaintext compromise.

### 6.2 Strong Key Hierarchy

Using a password-derived KEK to wrap a random AMK is the correct design. It enables password rotation, recovery wrapping, future device workflows, and scoped collection sharing without re-encrypting all vault data.

### 6.3 Explicit Device Enrollment

Refusing OAuth-only device enrollment is a major positive. Most failures in this category come from convenience-driven fallback paths.

### 6.4 Conservative Revocation Semantics

The design correctly states that revocation only protects future state. That honesty matters because many products imply stronger guarantees than their protocol can provide.

### 6.5 Thin Control Plane

A thin control plane narrows the blast radius of server compromise and reduces the temptation to centralize secret handling over time.

### 6.6 Append-Only and Immutable Object Bias

Immutable records are a good fit for security. They reduce accidental overwrite, simplify auditability, and make idempotent replay tractable.

## 7. Threat Surface by Component

### 7.1 Client Applications

Threats:

- offline password guessing against persisted state
- extraction of decrypted local cache or search index
- memory scraping while unlocked
- failure to wipe decrypted state on lock or sign-out
- import of malicious or malformed vault data
- clipboard leakage

Assessment:

The design acknowledges local risk but needs sharper implementation controls. Local search and offline support are product strengths, but they create one of the main confidentiality tradeoffs in the whole system.

### 7.2 Control Plane

Threats:

- issuing over-broad object-store access
- enrolling a device without proper proof
- serving stale or rolled-back metadata
- leaking sensitive metadata through logs and metrics
- abuse of invite coordination or membership state

Assessment:

The control plane is not trusted for confidentiality, but it is highly trusted for integrity and authorization. That makes it security-critical even if it never sees plaintext.

### 7.3 Object Storage

Threats:

- ciphertext exfiltration
- replay of stale mutable objects
- overwrite or deletion of current pointers
- enumeration of object keys and metadata
- inconsistency caused by partial writes

Assessment:

The plan recognizes these risks, especially around compare-and-swap and rollback detection. That is necessary. The design still needs strict rules for authenticity of mutable metadata and for namespace leakage.

### 7.4 Browser Extension

Threats:

- page JavaScript access to plaintext
- over-broad origin matching
- injection into the wrong frame or origin
- secret retention in DOM, memory, or logs
- abuse of extension messages by compromised pages

Assessment:

This is the highest-risk user-facing surface. The design treats it as such, which is correct, but v1 security depends on enforcing conservative defaults in practice.

### 7.5 Sharing and Invites

Threats:

- sending the wrong key to the wrong recipient
- stale membership state after revocation
- confused-deputy authorization across owner and shared namespaces
- accepting an invite on an unintended device

Assessment:

Sharing is the hardest feature after sync correctness. The plan correctly delays it until after the single-user core, which materially improves the odds of getting it right.

## 8. Security Findings

This section identifies the main issues in the current design and plan.

### 8.1 Critical: Mutable Metadata Authenticity Is Not Yet Strict Enough

Affected design areas:

- `config.json`
- `heads/latest.json`
- snapshot pointers
- device records
- membership status
- invite status

Why this matters:

In this design, a small amount of mutable metadata controls which immutable state becomes authoritative. If an attacker can replay or tamper with those mutable records, they may not decrypt data, but they can:

- roll clients back to stale state
- hide recent revocations
- suppress new devices or memberships
- direct clients to attacker-chosen snapshots or event heads
- induce divergence across devices

Current gap:

The system design says events may have an optional author signature. Optional is not sufficient for a zero-knowledge system whose control plane and storage are untrusted for integrity. The design mentions rollback detection, canonical bytes, and signed event/head format in places, but the requirement is not consistently elevated to mandatory protocol behavior.

Required action:

- Define mandatory authenticated canonical encoding for all security-relevant mutable metadata.
- Define which keys sign which records.
- Define freshness and rollback rules for account config, head pointers, snapshots, device records, and memberships.
- Make signature verification and monotonicity checks blocking behavior, not telemetry.

Release impact:

This is a release blocker.

### 8.2 Critical: Capability Issuance Could Defeat the Crypto Model if Path Scoping Is Loose

Affected design areas:

- signed URL issuance
- scoped object credentials
- shared collection access
- storage namespace design

Why this matters:

A zero-knowledge product can still fail if the server grants the wrong ciphertext to the wrong party. While ciphertext may remain unreadable in some cases, sharing flows, wrapped keys, metadata, and future writes are all governed by authorization scope.

Main failure modes:

- granting owner-account prefix access when only a collection path is needed
- allowing listing instead of direct object fetch
- using long-lived capabilities
- reusing credentials across device or membership state changes
- not binding capabilities to explicit HTTP methods, prefixes, and expiry

Required action:

- Define storage authorization at the object-prefix level with no owner-prefix overreach.
- Prefer direct object GET/PUT permissions rather than list privileges where possible.
- Use very short-lived credentials or signed URLs.
- Bind issued capabilities to account ID, device ID, collection ID, operation, and expiry.
- Test replay, prefix escape, method escalation, and revocation timing explicitly.

Release impact:

This is a release blocker.

### 8.3 High: Device Enrollment Integrity Needs Explicit Ceremony Binding

Why this matters:

Device enrollment is effectively root-of-trust expansion. If this path is ambiguous, an attacker who controls OAuth, the control plane, or a phishing flow may gain durable access without ever learning the master password.

Current strengths:

- OAuth-only enrollment is rejected.
- enrollment requires existing-device approval or recovery-key bootstrap.

Remaining gaps:

- the exact approval ceremony is not specified tightly enough
- there is no explicit requirement for user-visible verification of the new device
- there is no explicit anti-downgrade rule if one enrollment path is weaker than another
- device revocation semantics for existing cached capabilities are not fully specified

Required action:

- Bind enrollment approvals to a specific new-device public key, device label, timestamp, and nonce.
- Show a user-verifiable short code or fingerprint on both devices for existing-device approval.
- Require explicit confirmation on the approving device.
- Make recovery-key bootstrap at least as strong as existing-device enrollment for AMK unwrap.
- Define capability invalidation and re-auth requirements after device revocation.

Release impact:

This is a release blocker for multi-device support.

### 8.4 High: Invite Acceptance and Recipient Binding Are Under-Specified

Why this matters:

Sharing security depends on wrapping the right collection key to the right recipient and device context. If the recipient identity, account, or device binding is weak, keys can be misdelivered.

Risks:

- invite stolen or replayed before intended acceptance
- invite accepted under the wrong account
- collection key wrapped to a stale or attacker-inserted device key
- acceptance flow relying too heavily on control-plane assertions

Required action:

- Define whether invites are bound to account identity, device identity, or both.
- Bind acceptance to an authenticated recipient account and an explicit recipient device key.
- Treat invite status changes as authenticated state transitions.
- Define expiry, one-time-use semantics, and replay rejection.

Release impact:

This is a release blocker for sharing.

### 8.5 High: Local Decrypted Search Index Is a Confidentiality Hotspot

Why this matters:

The local search index contains decrypted derivatives of the most sensitive user data. For many users, it is effectively a plaintext catalog of the vault.

Risks:

- disk extraction on desktop
- backup tooling capturing index contents
- stale index persistence after sign-out
- extension environment retaining searchable metadata longer than intended

Required action:

- Treat the search index as high-sensitivity material equal to plaintext metadata.
- Encrypt it at rest using keys only available while unlocked or re-derived securely.
- Wipe it on sign-out and provide tested rebuild logic.
- Minimize indexed fields in the extension specifically.

Release impact:

High. Not necessarily a blocker if the local encryption model is sound, but it must be implemented deliberately and tested adversarially.

### 8.6 High: Browser Extension Boundary Is the Most Likely Plaintext Leak

Why this matters:

Extensions operate in hostile territory. A correct vault core does not save an extension that fills the wrong page, leaks via logs, or exposes secrets through message-passing.

Required controls:

- all cryptographic operations and secret access remain in background or service-worker context
- content scripts get only narrowly scoped plaintext at fill time
- strict origin and frame matching
- explicit handling of redirects, iframes, subdomains, and public-suffix pitfalls
- no automatic fill on ambiguous matches
- hard production prohibition on sensitive logs

Release impact:

This is a release blocker for the extension.

### 8.7 High: Metadata Minimization Is Good in Principle but Incomplete in Practice

Why this matters:

The design correctly acknowledges metadata leakage, but the storage layout still exposes meaningful structure:

- account IDs and collection IDs in object paths
- event cadence and object count
- item version churn
- timestamps and size variation
- shared collection structure

This may be acceptable for v1, but it should be treated as a conscious privacy tradeoff, not a solved problem.

Required action:

- document exact path and object-name leakage
- avoid human-meaningful names in keys
- review whether IDs are random, opaque, and non-enumerable
- ensure logs and metrics do not reintroduce metadata leakage at higher fidelity than storage already does

Release impact:

High, but likely acceptable with explicit documentation and logging discipline.

### 8.8 Medium: Offline Password Guessing Resistance Depends on Parameter Governance, Not Just Argon2id

Why this matters:

Argon2id is the right choice, but naming the primitive is not enough. The actual memory and time cost, upgrade strategy, and device-class tuning determine resistance in practice.

Required action:

- fix concrete Argon2id parameters for target device classes
- version parameters in account config
- add re-parameterization strategy over time
- include low-memory and abuse-case testing

Release impact:

Important but tractable.

### 8.9 Medium: Export and Import Are Major Exfiltration and Integrity Surfaces

Why this matters:

Export exists to move secrets across trust boundaries. Import exists to ingest attacker-controlled structured data. Both are dangerous.

Required action:

- define secure export formats and whether exports may be encrypted
- require explicit user acknowledgement for plaintext export
- sanitize filenames and metadata
- validate imported schemas strictly
- protect against oversized payloads, compression bombs, and malicious field content

Release impact:

High for product risk, medium for architectural risk.

### 8.10 Medium: No Primary Database Simplifies Confidentiality but Makes Certain Integrity and Audit Problems Harder

Why this matters:

Avoiding a traditional database is sensible for v1, but it removes easy server-side mechanisms for:

- durable audit indexing
- revocation state coordination
- rate-limit correlation
- invite lifecycle introspection

The control plane must still hold enough durable state to make correct authorization decisions. If that state is recreated ad hoc or inconsistently cached, security will drift.

Required action:

- define the minimal durable control-plane state clearly
- separate authorization metadata from vault content storage without pretending no state exists
- ensure revocation and invite status are strongly consistent enough for their purpose

Release impact:

Medium. This is mainly a precision issue in the design.

## 9. Assessment of Major Design Areas

### 9.1 Cryptography

Assessment:

Strong overall direction. The hierarchy is correct, the primitive families are appropriate, and password rotation is well designed.

Required clarifications:

- choose one primary AEAD suite rather than deferring indefinitely
- specify nonce generation and uniqueness rules
- specify canonical encoding for signed content
- specify key versioning, migration, and deprecation rules
- specify secure random source requirements across all clients

Conclusion:

The cryptographic design is sound enough for v1 if protocol authenticity and parameterization are nailed down.

### 9.2 Sync and Object Storage

Assessment:

This is the hardest backend problem in the system. The design is good because it recognizes compare-and-swap head updates, immutable object pre-write, idempotency keys, and partial-failure recovery as first-class concerns.

Main risk:

Silent acceptance of stale or forged mutable state.

Conclusion:

This area is viable, but only with authenticated mutable state, deterministic replay, and strong fault-injection testing.

### 9.3 Sharing and Revocation

Assessment:

The collection-scoped model is the right v1 simplification. The design is honest about revocation limitations and correctly separates authorization revocation from cryptographic forward revocation.

Main risk:

Misbinding of recipients or storage authorization.

Conclusion:

Viable for v1 if invite and membership state transitions are tightly specified and tested.

### 9.4 Recovery

Assessment:

The recovery model is consistent with zero knowledge and should be part of the core architecture, not a late-stage UX feature.

Concern:

The implementation plan treats recovery as both a must-ship capability and something refined late. That is acceptable only if the underlying cryptographic recovery path is proven early, which the plan mostly says, but the sequencing should stay disciplined.

Conclusion:

Good design. Must be validated early and repeatedly.

### 9.5 Extension

Assessment:

Appropriately treated as high risk.

Concern:

Extensions routinely fail due to edge cases in origin binding, frame matching, redirect flows, and plaintext lifecycle rather than due to core cryptography.

Conclusion:

Only ship with conservative defaults and adversarial testing.

## 10. Required Security Controls

The following controls should be treated as required for v1.

### 10.1 Protocol and Integrity Controls

- authenticated canonical serialization for all security-relevant records
- mandatory signature or equivalent authenticity verification for mutable metadata
- monotonic rollback detection for heads, snapshots, memberships, and account config
- idempotency keys for writes
- compare-and-swap commit point enforcement

### 10.2 Authorization Controls

- object access scoped to exact prefixes or objects
- no broad owner-prefix delegation for shared collection access
- short-lived signed URLs or scoped credentials
- capability issuance bound to account, device, operation, and expiry
- explicit invalidation behavior after revocation

### 10.3 Client Hardening Controls

- secure local storage for device private keys where platform support exists
- encrypted local cache and encrypted or unlock-bound decrypted index
- reliable memory and cache wipe on lock and sign-out where practical
- no plaintext in logs, traces, crash reports, analytics, or debug tooling
- secure import/export handling

### 10.4 Extension Controls

- background-only secret boundary
- narrow message API between content scripts and background context
- strict origin and frame matching
- conservative autofill default
- clipboard actions explicit and time-limited

### 10.5 Operational Controls

- bucket versioning enabled
- lifecycle controls for abandoned writes
- audit events for enrollment, invite, revocation, and capability issuance
- telemetry reviewed for secret and metadata leakage
- incident runbooks for metadata exposure and rollback/tampering incidents

## 11. Verification Strategy

The implementation plan already includes many of the right tests. The assessment adds emphasis on the following.

### 11.1 Must-Have Automated Tests

- cross-platform crypto test vectors
- wrong-password, wrong-recovery-key, and corrupted-ciphertext tests
- stale-head, stale-snapshot, and stale-membership replay rejection
- concurrent writer tests with forced CAS contention
- capability scope tests for prefix escape and method escalation
- device enrollment tests with mismatched or substituted device keys
- invite replay and account/device misbinding tests
- extension tests for iframe confusion, subdomain confusion, redirect confusion, and page-script isolation
- local lock/sign-out tests validating removal of decrypted derivatives

### 11.2 Must-Have Adversarial Review

- protocol review of signed mutable state and rollback handling
- storage authorization review
- extension threat-model review
- logging and telemetry redaction review
- recovery flow walkthrough using persisted fixtures only

### 11.3 Manual Exercises

- simulated revoked-device access attempt using cached capabilities
- simulated shared-member revocation followed by future-write access attempt
- local disk inspection of a locked desktop client
- extension fill attempts against intentionally deceptive domains

## 12. Release Blockers

The following should block a public v1 release if unresolved.

1. No mandatory authenticity and rollback protection for mutable metadata.
2. No proof that storage capabilities are tightly path-scoped and short-lived.
3. No fully specified and tested device enrollment ceremony.
4. No fully specified and tested invite acceptance and recipient binding model.
5. No extension security review with origin-binding and page-isolation validation.
6. No redaction review for logs, telemetry, crash reporting, and debug tooling.
7. No demonstrated recovery flow using serialized persisted account state.

## 13. Residual Risks Acceptable for v1

The following residual risks are acceptable if communicated clearly.

- metadata leakage from counts, sizes, timing, and object structure
- inability to protect an already-compromised unlocked endpoint
- inability to retroactively revoke previously seen plaintext
- user error in handling exported plaintext
- some platform variability in secure local key storage

These are acceptable only if product and support language are precise and do not imply stronger guarantees.

## 14. Recommended Changes to the Existing Plan

The current implementation plan is good, but the following adjustments would improve security execution.

### 14.1 Move Integrity Design From Implicit to Explicit Phase 0 Output

Phase 0 should explicitly freeze:

- the authenticity model for mutable metadata
- signing keys and verification rules
- rollback-detection markers and monotonic counters

This is too important to leave partially emergent during sync implementation.

### 14.2 Treat Recovery as Core, Not Late Hardening

The cryptographic recovery path should be complete in Phase 1 and exercised continuously. Phase 8 should refine UX and operations, not validate the existence of the security model for the first time.

### 14.3 Add a Dedicated Capability-Scoping Review Before Sharing

Before Phase 7, add a formal review of:

- account-path credentials
- shared-collection credentials
- revocation timing
- prefix isolation

### 14.4 Add an Import/Export Security Gate

Export and import need their own gate:

- plaintext export warnings
- encrypted export format decision
- strict import validation
- large-input and malformed-input testing

### 14.5 Add a Local Data Exposure Review

Before desktop MVP and extension release, review:

- decrypted search index contents
- cache retention while locked
- backup interactions
- crash dump exposure

## 15. Final Assessment

Safe has a credible security architecture for a zero-knowledge secret manager. The design makes several disciplined choices that many systems avoid: explicit device enrollment, client-only key access, forward-looking revocation semantics, and a thin control plane.

The biggest unresolved risk is not whether the service can decrypt data. The biggest unresolved risk is whether the system can preserve integrity and authorization in the face of an untrusted control plane and object store while still delivering offline sync, sharing, and extension-based autofill.

If the team makes authenticated mutable metadata, capability scoping, enrollment integrity, extension isolation, and operational redaction first-class release criteria, the design can support a strong v1. If those remain partially specified or are postponed behind product surface work, the system will be materially weaker than its zero-knowledge positioning suggests.
