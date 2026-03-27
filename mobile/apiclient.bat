@echo off
setlocal

if not exist ..\server\docs\swagger.json (
  echo Missing ..\server\docs\swagger.json
  echo Regenerate the server spec from the repo root first.
  exit /b 1
)

copy /Y ..\server\docs\swagger.json .\schema\swagger.json >nul
dart run swagger_parser -f .\swagger_parser.yaml || exit /b 1
dart run build_runner build --delete-conflicting-outputs || exit /b 1
flutter gen-l10n || exit /b 1
