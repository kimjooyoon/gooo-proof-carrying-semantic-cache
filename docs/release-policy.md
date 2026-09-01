# Release policy

The program has zero authority for repository writes, commits, pushes, merges,
or releases. Pull request creation, merge, annotated tag creation, immutable
release publication, and release verification are human-authorized CI/release
operations.

Before release, the pull request must pass the semantic audit and the GitHub
Actions workflow on the exact merge candidate. Release automation must refuse
an existing tag or release, use a new annotated tag pointing to the merged
commit, attach digests for the source and evidence assets, and verify the public
release API reports `immutable=true`. A failed or non-immutable release attempt
is retained; its tag or release is never deleted, edited, or overwritten.

The repository setting is enabled once with the user-authenticated
`scripts/enable-immutable-releases.sh OWNER/REPOSITORY` operation. The release
workflow does not assume that its ordinary Actions token has administration
scope; it verifies the published release object instead.
