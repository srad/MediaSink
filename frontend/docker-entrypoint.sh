#!/bin/sh

cat >/usr/share/nginx/html/env.js <<EOL
window.APP_APIURL = "${APP_API_URL}";
window.APP_BASE = "${APP_BASE}";
window.APP_NAME = "${APP_NAME}";
window.APP_SOCKETURL = "${APP_SOCKET_URL}";
window.APP_FILEURL = "${APP_FILE_URL}";
EOL

nginx -g "daemon off;"
