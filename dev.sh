#!/usr/bin/env bash
# คำสั่งที่ใช้บ่อยของโปรเจกต์ — ./dev.sh help
set -euo pipefail
cd "$(dirname "$0")"

case "${1:-help}" in
  up)    docker compose up -d --build ;;                      # สร้างและรันทั้งระบบ
  down)  docker compose down ;;                               # หยุดระบบ (ข้อมูลยังอยู่)
  logs)  docker compose logs -f api web ;;                    # ดู log ของ api และ web
  reset) docker compose run --rm seed --reset ;;              # ล้างข้อมูลแล้วใส่ข้อมูลตัวอย่างใหม่
  lint)
    (cd api && go vet ./... && test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; })
    (cd web && npm run lint && npx tsc --noEmit)
    ;;
  test)                                                       # unit + integration (Go ต้องมี Docker สำหรับ testcontainers)
    (cd api && go test ./...)
    (cd web && npm test)
    ;;
  e2e)                                                        # E2E กับระบบที่รันอยู่ — ล้าง seed ก่อนให้ผลคงที่
    shift
    docker compose run --rm seed --reset
    docker compose --profile e2e run --rm -e E2E_ARGS="$*" e2e
    ;;
  check) "$0" lint && "$0" test && "$0" e2e ;;                 # ตรวจครบทุกอย่างก่อนส่งงาน
  *)
    cat <<'HELP'
./dev.sh <คำสั่ง>
  up      สร้างและรันทั้งระบบ (http://jongyoung.localhost)
  down    หยุดระบบ
  logs    ดู log ของ api และ web
  reset   ล้างข้อมูลแล้วใส่ข้อมูลตัวอย่างใหม่
  lint    go vet + gofmt + eslint + tsc
  test    unit/integration test ของ Go และ web
  e2e     E2E test (Playwright) กับระบบที่รันอยู่ — ส่งต่ออาร์กิวเมนต์ได้ เช่น ./dev.sh e2e --project=mobile
  check   lint + test + e2e
HELP
    ;;
esac
