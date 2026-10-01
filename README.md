# doc-server-go

Prosty, samodzielny serwer dokumentacji Markdown dla projektu **DTR Żarna (PAP + BNS)**.

## Co robi

- Serwuje stronę główną z `start.md` i opcjonalną legendą z `legenda.md`.
- Wyświetla dokumentację z folderu `dokumenty/` z podziałem na podfoldery.
- Pozwala zalogowanym użytkownikom domenowym (LDAP) edytować pliki Markdown z blokadą per plik.
- Ogranicza klikanie checkboxów na stronie startowej do wybranej listy użytkowników z konfiguracji.
- Odświeża stronę startową, gdy `start.md` lub `legenda.md` zmienią się na dysku.
- Próbuje automatycznie kolejnych portów, jeśli domyślny jest zajęty.

## Struktura

```text
doc-server-go/
  main.go                  # kod serwera (Go)
  go.mod / go.sum          # moduły Go
  ustawienia.toml          # konfiguracja lokalna (nie wrzucać do repo!)
  ustawienia.toml.example  # szablon konfiguracji
  start.md                 # treść strony głównej (lokalna, ignorowana w Git)
  start.md.example         # przykład strony głównej
  legenda.md               # opcjonalna legenda pod start.md (lokalna, ignorowana w Git)
  legenda.md.example       # przykład legendy
  dokumenty-example/       # przykładowe drzewo dokumentacji (do repo)
    PAP/DTR/               # przykładowe dokumenty maszyny PAP
    BNS/DTR/               # przykładowe dokumenty maszyny BNS
    Wspolne/               # przykładowe dokumenty wspólne
  dokumenty/               # realne drzewo dokumentacji (lokalnie, ignorowane w Git)
  static/                  # zasoby statyczne (EasyMDE, CSS)
  uruchom-serwer.bat       # uruchomienie w tle z logami
  zabij-serwer.bat         # zatrzymanie wszystkich instancji
  install-service.bat      # rejestracja autostartu przez Harmonogram zadań
  zadanie.xml              # definicja zadania dla Harmonogramu
```

## Wymagania

- Go 1.23+
- Windows (dla skryptów `.bat` / Harmonogramu zadań)
- Dostęp do kontrolera domeny LDAP, jeśli używasz logowania domenowego

## Przygotowanie dokumentacji

```bash
cp -r dokumenty-example dokumenty
cp start.md.example start.md
cp legenda.md.example legenda.md
# albo w Eksploratorze: skopiuj foldery/pliki .example jako oryginały
```

Następnie uzupełnij lub zastąp pliki w `dokumenty/`, `start.md` i `legenda.md` właściwą treścią.

## Konfiguracja

```bash
cp ustawienia.toml.example ustawienia.toml
# albo w Notatniku: ustawienia.toml.example -> ustawienia.toml
```

Uzupełnij:

- `port` – port HTTP (domyślnie 8080).
- `ldap_url`, `domain`, `domain_nt` – dane kontrolera domeny.
- `[users]` – loginy domenowe, które mogą przełączać checkboxy na stronie startowej.

## Budowa

```bash
go build -o doc-server-go.exe .
```

## Uruchamianie

### Ręcznie

```bash
./doc-server-go.exe
```

lub skryptem Windows:

```batch
uruchom-serwer.bat
```

### Autostart systemu Windows

Uruchom jako Administrator:

```batch
install-service.bat
```

Zatrzymanie/usunięcie:

```batch
zabij-serwer.bat
schtasks /Delete /TN "DocServerGo" /F
```

## Użytkowanie

- Strona główna: `http://<host>:8080/`
- Lista dokumentów: `http://<host>:8080/docs`
- Edycja: zaloguj się domenowo, wejdź w dokument i kliknij **Edytuj**.
- Tylko jedna osoba może edytować dany plik naraz; blokada widoczna jest dla innych użytkowników.

## Bezpieczeństwo

Plik `ustawienia.toml` zawiera dane wewnętrznej sieci (IP kontrolera, domena, lista loginów)
i jest ignorowany przez Git. Nie wysyłaj go na GitHub. Do repo przeznaczony jest plik
`ustawienia.toml.example`.

## Licencja

Wewnętrzny projekt firmowy – do użytku wewnętrznego.
