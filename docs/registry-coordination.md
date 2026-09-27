# Managed Registry coordination

`RbdComponent.spec.registryCoordination` is an opt-in configuration for `rbd-hub`.
Without it, existing Registry containers and Service ports are unchanged. This
configuration is a deployment prerequisite, not a declaration that deletion or
GC is ready.

An enabled configuration needs:

- Digest-pinned native Registry and combined coordinator/GC images. The latter
  must come from a bootstrap source independent of the Registry being managed.
- Core-provided `storageID`, `generation`, `volumeUID`, and `registryPath`.
- A Console origin, enterprise and region scope; internal HTTP is explicit.
- Separate `controlSecret` and `permitSecret`, each projecting the `key` entry.
  Secret values must not be written into the RbdComponent.
- The Core API independently mounted to the same permit-signing Secret.

The Operator generates a storage-identity initializer, a read-only coordinator
sidecar, loopback-only native Registry listening, disabled legacy upload purging,
coordinated readiness probes, and public Service port 5000 targeting port 5001.
Recreate rollout avoids overlapping uncoordinated old Pods. Only zero or one
replica is supported for this installation. Resources can be supplied through
`registryCoordination.resources`.

The initializer only creates/verifies this installation's storage identity; it
does not run GC. GC remains a separate Core-generated, explicitly confirmed Job.
The sidecar uses its installation credential for control requests and an
independent permit key for deletion verification. GC does not receive the latter.

## Rollout status and boundaries

Do not switch a live Registry merely by applying these fields. Complete the
coordinator image, Core API configuration, stable control-credential lifecycle,
producer coverage, ingress review and recovery prerequisites first. In particular,
the control credential must remain resolvable independently of plugin removal.
That lifecycle integration and the controlled deactivation/migration procedure
are separate rollout prerequisites, not supplied by this configuration alone.

Clearing the field while an existing coordinated Deployment is present is
rejected. This prevents an ordinary reconciliation from exposing an uncoordinated
native port during outstanding operations. There is no force-disable flag.
Do not downgrade the Operator or CRD while coordination is installed.

## Generation and validation

The generator is pinned to controller-tools v0.16.5, compatible with the Go 1.22
module and the development Go toolchain. The previously missing boilerplate header
is included. Generated CRDs preserve all prior validation constraints; the new
configuration is optional.

Run the isolated Go tests, `go vet ./...`, and `go build ./...`. Tests must not use
a developer's kubeconfig or an existing cluster.
