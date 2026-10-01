# doc-server-go

Prosty serwer dokumentacji Markdown dla projektu DTR Żarna (maszyny PAP i BNS).

## Pierwsze uruchomienie

1. Sklonuj lub rozpakuj folder `doc-server-go`.
2. Skopiuj pliki przykładowe i uzupełnij je właściwą treścią:

   ```bash
   cp start.md.example start.md
   cp legenda.md.example legenda.md
   cp -r dokumenty-example dokumenty
   cp ustawienia.toml.example ustawienia.toml
   ```

3. Otwórz `ustawienia.toml` i wpisz:
   - `port` – port HTTP (domyślnie `8080`),
   - dane kontrolera domeny LDAP (`ldap_url`, `domain`, `domain_nt`),
   - loginy domenowe, które mogą przełączać checkboxy na stronie głównej (`[users]`).

4. Pobierz gotową binarkę z zakładki **Releases** na GitHubie lub zbuduj serwer samodzielnie:

   ```bash
   go build -o doc-server-go.exe .
   ./doc-server-go.exe
   ```

   Szczegóły budowy, skrypty Windows, autostart oraz proces tworzenia wydań znajdziesz w [`build.md`](build.md).

5. Otwórz w przeglądarce:

   ```text
   http://localhost:8080
   ```

## Korzystanie

- **Strona główna** (`/`) – wyświetla `start.md` i `legenda.md`; odświeża się automatycznie po ich zapisaniu na dysku.
- **Lista dokumentów** (`/docs`) – przeglądanie drzewa `dokumenty/` z podziałem na podfoldery.
- **Podgląd dokumentu** (`/view`) – wyświetla Markdown, status edycji i obecnych użytkowników.
- **Edycja** – wymaga zalogowania loginem domenowym. Tylko jedna osoba może edytować dany plik naraz; inni widzą, kto trzyma blokadę.
- **Checkboxy na stronie głównej** – mogą je przełączać tylko użytkownicy wpisani w `ustawienia.toml`.

Pliki `start.md`, `legenda.md`, `ustawienia.toml` oraz cały folder `dokumenty/` są ignorowane przez Git – pozostają lokalnie.
