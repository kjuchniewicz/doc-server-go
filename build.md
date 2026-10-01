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

## Wydania (GitHub Releases)

Projekt wykorzystuje GitHub Actions do automatycznego budowania binarki Windows i
publikowania jej jako Release.

### Automatyczne wydanie po tagu

1. Upewnij się, że wszystkie zmiany są w głównej gałęzi i działają poprawnie.
2. Utwórz i wypchnij tag zgodny z Semantic Versioning:

   ```bash
   git tag -a v1.0.0 -m "Pierwsze stabilne wydanie"
   git push origin v1.0.0
   ```

3. GitHub Actions (`.github/workflows/release.yml`) automatycznie:
   - zbuduje binarkę `doc-server-go.exe` na Windows,
   - utworzy Release z nazwą tagu,
   - dołączy plik `doc-server-go.exe` jako załącznik.

### Ręczne wydanie

Jeśli nie chcesz korzystać z Actions, możesz zbudować binarkę lokalnie:

```bash
go build -ldflags "-s -w" -o doc-server-go.exe .
```

Następnie w GitHubzie:

- **Releases → Draft a new release**,
- wybierz lub utwórz tag,
- nazwij wydanie i dodaj opis,
- przeciągnij `doc-server-go.exe` do pola załączników,
- opublikuj.

### Co powinno zawierać wydanie

- Binarka `doc-server-go.exe`.
- Pliki konfiguracyjne / przykłady z repo (`ustawienia.toml.example`,
  `start.md.example`, `legenda.md.example`, `dokumenty-example/`),
  które użytkownik musi skopiować i uzupełnić przed uruchomieniem.

> Uwaga: nie dołączaj `ustawienia.toml`, `start.md`, `legenda.md` ani folderu
> `dokumenty/` – zawierają one lokalne, często wrażliwe dane projektu.
