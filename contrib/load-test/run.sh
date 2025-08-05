#!/bin/sh -e

HOSTNAME="dg-server"

[ -z "$LOAD_TEST_DIR" ] && (echo "LOAD_TEST_DIR is not set"; exit 1)
cd $(dirname $(readlink -f $0))

../gen-certs.sh --hostname ${HOSTNAME} --data-dir "${LOAD_TEST_DIR}" --num-devices 3

[ -z ${NOBUILD} ] && docker compose build

cat > ${LOAD_TEST_DIR}/fake-devices/device-1/urls <<EOF
GET https://${HOSTNAME}:8443/device
EOF

cat > ${LOAD_TEST_DIR}/fake-devices/device-2/urls <<EOF
GET https://${HOSTNAME}:8443/device
EOF

cat > ${LOAD_TEST_DIR}/fake-devices/device-3/urls <<EOF
GET https://${HOSTNAME}:8443/device
EOF

echo "## Running load test for 15 seconds..."
# all the /dev/null stuff helps keep compose from trying to do fancy tty stuff
timeout -v --foreground -sTERM -k17s 15s docker compose up < /dev/null | tee /dev/null

docker run --rm -v ${LOAD_TEST_DIR}:/results load-test-vegeta-1 \
    report \
    /results/fake-devices/device-1/results.bin \
    /results/fake-devices/device-2/results.bin \
    /results/fake-devices/device-3/results.bin > ${LOAD_TEST_DIR}/results.txt

cat ${LOAD_TEST_DIR}/results.txt
