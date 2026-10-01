# Budowa i rozwój serwera

## Wymagania

- Go 1.23 lub nowszy
- Windows (skrypty `.bat` / Harmonogram zadań)
- Dostęp do kontrolera domeny LDAP, jeśli używasz logowania domenowego

## Zależności

Zależności są zarządzane przez moduły Go (`go.mod` / `go.sum`):

- `github.com/go-ldap/ldap/v3` – logowanie przez LDAP / Active Directory
- `github.com/gomarkdown/markdown` – renderowanie Markdown do HTML

## Budowa

```bash
go build -o doc-server-go.exe .
```

Powstaje plik wykonywalny `doc-server-go.exe` w bieżącym folderze.

## Uruchamianie

### Ręcznie

```bash
./doc-server-go.exe
```

Serwer próbuje portu z `ustawienia.toml` (domyślnie `8080`). Jeśli port jest zajęty,
próbuje kolejnych do `port + 4`.

### Skryptem Windows

```batch
uruchom-serwer.bat
```

Uruchamia serwer w tle i zapisuje logi do `serwer.log` / `serwer-err.log`.

Zatrzymanie:

```batch
zabij-serwer.bat
```

## Autostart systemu Windows

Uruchom jako Administrator:

```batch
install-service.bat
```

Rejestruje zadanie w Harmonogramie zadań (`DocServerGo`), które startuje serwer
przy restarcie systemu.

Zatrzymanie i usunięcie:

```batch
schtasks /End /TN "DocServerGo"
schtasks /Delete /TN "DocServerGo" /F
```

## Struktura projektu

```text
doc-server-go/
  main.go                  # kod źródłowy serwera
  go.mod / go.sum          # moduły Go
  static/                  # EasyMDE, CSS, zasoby statyczne
  uruchom-serwer.bat       # start w tle
  zabij-serwer.bat         # zatrzymanie
  install-service.bat      # instalacja autostartu
  zadanie.xml              # definicja zadania Harmonogramu
```

## Rozwój i debugowanie

- `go run .` – uruchamia serwer bez budowania pliku wykonywalnego.
- `go test ./...` – uruchamia testy (jeśli są dodane).
- Logi bieżącej sesji pojawiają się w `serwer.log` / `serwer-err.log`.
