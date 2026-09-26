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
  docs)                                                       # สร้าง Swagger 2 (swaggo) จาก comment ใน handler แล้วแปลงเป็น OpenAPI 3
    (cd api && swag init -g cmd/api/main.go -o docs --parseInternal)
    npx -y swagger2openapi@7 --yaml -o api/docs/openapi.yaml api/docs/swagger.yaml
    # Swagger 2 ไม่มีชนิด bearer (swaggo จึงใช้ apiKey) — OpenAPI 3 มี จึงเปลี่ยนให้ถูกชนิด
    sed -i '/^    BearerAuth:/,/^      type: apiKey/c\    BearerAuth:\n      description: access token จาก Keycloak (realm jongyoung, aud = jongyoung-api)\n      type: http\n      scheme: bearer\n      bearerFormat: JWT' api/docs/openapi.yaml
    sed -i '1i # สร้างด้วย ./dev.sh docs จาก comment ใน handler — ห้ามแก้ไฟล์นี้ด้วยมือ' api/docs/openapi.yaml
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
  docs    สร้าง api/docs/swagger.yaml + openapi.yaml ใหม่จาก comment ใน handler
  check   lint + test + e2e
HELP
    ;;
esac
