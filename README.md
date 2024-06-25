# Wiki Telegram bot

Wiki Telegram bot - это телеграм-бот для поиска информации в Wikipedia.
Пользователь отправляет запрос боту, на какую тему он хочет получить информацию, после чего получает статьи из Wikipedia на эту тему.

### Доступные команды:

**/start** - стандартная команда, при первом обращении к боту (приветствует и описание информации о боте)
**/info** - информация о боте и командах
**/request-history** - получение информации о последних 3 запросах
**/response-history** - получение информации о последних 3 полученных ответов на запрос

Попробовать: https://t.me/wiki_pedia_tg_bot
Если ссылка не открывается, в этом случае откройет телеграм и введите в поиске wiki_pedia_tg_bot

### Сборка репозитория и локальный запуск

Выполните в консоли

```Console
git clone https://github.com/oN1mode/WikiTG.git
```

Затем перейти в корневую папку проекта

```Console
cd WikiTG
```

### Настройка

Создайте файл .env и добавьте следующие настройки

```
    BOT_API_TOKEN="Token telegram bot father"

    HOST="database host"
    PORT="port"
    USER="name database user"
    PASSWORD="database password"
    DBNAME="database name"
    SSLMODE="database ssl mode"
```

### Запуск

Чтобы запустить бота, выполните в консоли команду:

```Console
go run cmd/app/main.go
```
