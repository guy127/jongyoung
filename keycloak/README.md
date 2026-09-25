# Keycloak realm `jongyoung`

`import/jongyoung-realm.json` ถูกโหลดอัตโนมัติตอน `docker compose up` (`start-dev --import-realm`)
ถ้า realm มีอยู่แล้ว Keycloak จะข้ามการ import — แก้ใน Admin Console ได้โดยไม่ถูกทับ

## มีอะไรใน realm

| รายการ | ค่า |
|---|---|
| client | `jongyoung-web` (confidential, standard flow; direct access grant เปิดไว้เพื่อทดสอบด้วย curl ตอน dev) |
| client secret | `dev-only-jongyoung-web-secret` — **ค่า dev เท่านั้น** ตรงกับ `.env.example` |
| audience mapper | access token มี `aud = jongyoung-api` (Go ตรวจค่านี้) |
| บัญชีทดสอบ | `owner1` / `owner1-pass`, `owner2` / `owner2-pass`, `customer1` / `customer1-pass` |

`id` ของผู้ใช้ทดสอบกำหนดตายตัว (`...0001`, `...0002`, `...0003`) เพื่อให้ seed data ผูก `users.keycloak_uid` ได้

## ขอ token ทดสอบด้วย curl

```bash
curl -s -X POST http://keycloak.jongyoung.localhost/realms/jongyoung/protocol/openid-connect/token \
  -d grant_type=password -d client_id=jongyoung-web -d client_secret=dev-only-jongyoung-web-secret \
  -d username=customer1 -d password=customer1-pass -d scope=openid
```

## Export ใหม่หลังแก้ใน Admin Console

```bash
docker compose exec keycloak /opt/keycloak/bin/kc.sh export \
  --dir /tmp/export --realm jongyoung --users realm_file
docker compose cp keycloak:/tmp/export/jongyoung-realm.json keycloak/import/jongyoung-realm.json
```
ตรวจ diff ก่อน commit — ต้องไม่มี secret จริงหลุดเข้าไป

## เริ่ม realm ใหม่ทั้งหมด

Keycloak ในโหมด dev เก็บข้อมูลในคอนเทนเนอร์ → `docker compose rm -sf keycloak && docker compose up -d keycloak`
