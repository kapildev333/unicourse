# P17-L4-review - review of P17-L4 (upload limit from env, GET /upload-policy, web reads it)
Status: done
## Doing
Review finished: approve, no blocking findings.
## Found
- Server matches design §6.2-§6.3 line for line; single allowlist source (`fileTypes` keys, sorted) for upload and policy; openapi entry inherits global `session` security, has 401/500 and the `UploadPolicy` schema; generated types/postman in sync.
- Real HTTP (unicourse_e2e_19, 127.0.0.1:17114, UNICOURSE_MAX_UPLOAD_BYTES=1048576): policy 401 signed out, 200 `{"maxBytes":1048576,...16 sorted}` signed in; uploads 1 MiB 201, 1 MiB+1 400, 3 MiB 400, chunked 3 MiB 400, Content-Length understated 400 (unexpected EOF), overstated 400 "is larger than 1 MB"; no files under .tmp afterwards.
- Nits (optional): students also fetch /upload-policy on Materials (materials.tsx:43, once per session); the MaxBytesReader branch (body > limit + 1 MiB in non-file parts) has no small-limit unit test (pre-existing path, only message changed).
- Request from P17-L1 for §8.2 openapi overlap text: already present in openapi.json (section create and import descriptions).
## Requests
none
## Blockers
none
