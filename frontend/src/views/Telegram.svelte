<script lang="ts">
    import { onMount } from 'svelte';
    import {
        Button,
        Card,
        DateField,
        Icon,
        SelectOutlined,
        TextField,
        TextFieldOutlinedMultiline,
        Dialog,
    } from "m3-svelte";
    import addIcon from "@ktibow/iconset-material-symbols/add-box-rounded";
    import sendIcon from "@ktibow/iconset-material-symbols/send-outline";

    // Импорт Wails связок
    import { 
        GetTgBots, 
        SaveTgBot, 
        GetTgChannels, 
        SaveTgChannel, 
        GetTgTemplates, 
        SaveTgTemplate, 
        SendToTelegram 
    } from '../../wailsjs/go/main/App';
    import { database } from '../../wailsjs/go/models';

    // Импортируем данные из стора для magic-переменных
    import { chapterTitle, finalUrl } from "../stores/appStore.js";

    // Состояние данных
    let bots: database.TgBot[] = [];
    let channels: database.TgChannel[] = [];
    let templates: database.TgTemplate[] = [];

    // Выбранные значения
    let selectedBotId: any = null; // ID выбранного бота
    let selectedChannelId: any = null; // ID выбранного канала (в базе)
    let selectedTemplateId: any = null;
    
    // Данные для постов
    let postText = "";
    let postDate = ""; // Формат YYYY-MM-DD
    let postTime = ""; // Формат HH:MM

    // Данные для модальных окон
    let openAddBot = false;
    let newBotToken = "";
    let newBotName = "My Bot"; 

    let openAddChanel = false;
    let newChannelLink = "";

    let openAddTemplate = false;
    let newTemplateName = "";

    // --- Инициализация ---
    onMount(async () => {
        await refreshAll();
    });

    async function refreshAll() {
        try {
            bots = await GetTgBots() || [];
            channels = await GetTgChannels() || [];
            templates = await GetTgTemplates() || [];
        } catch (e) {
            console.error("Ошибка загрузки данных:", e);
        }
    }

    // --- Вычисляемые свойства для Select ---
    
    // Преобразуем ботов в формат для Select
    $: botOptions = bots.map(b => ({ text: b.name || "Bot " + b.id, value: b.id }));

    // Фильтруем каналы по выбранному боту и преобразуем
    $: filteredChannelOptions = channels
        .filter(c => selectedBotId ? c.bot_id === selectedBotId : true)
        .map(c => ({ text: c.title || c.channel_id, value: c.channel_id })); 

    // Шаблоны
    $: templateOptions = templates.map(t => ({ text: t.name, value: t.id }));

    // --- Логика выбора шаблона с заменой переменных ---
    function onTemplateSelect() {
        const t = templates.find(temp => temp.id === selectedTemplateId);
        if (t) {
            let content = t.content;

            // Замена magic variables на значения из Store
            // Поддерживаем {{article_name}} и {{title}}
            content = content.replace(/{{article_name}}/g, $chapterTitle || "")
                             .replace(/{{title}}/g, $chapterTitle || "");
            
            // Поддерживаем {{article_url}} и {{url}}
            content = content.replace(/{{article_url}}/g, $finalUrl || "")
                             .replace(/{{url}}/g, $finalUrl || "");

            postText = content;
        }
    }

    // --- Логика сохранения (Handlers) ---

    async function handleSaveBot() {
        if (!newBotToken) return;
        const bot = new database.TgBot({
            token: newBotToken,
            name: newBotName 
        });
        await SaveTgBot(bot);
        newBotToken = "";
        openAddBot = false;
        await refreshAll();
    }

    async function handleSaveChannel() {
        if (!newChannelLink || !selectedBotId) {
            alert("Выберите бота и введите ссылку/ID канала");
            return;
        }
        const channel = new database.TgChannel({
            bot_id: selectedBotId,
            channel_id: newChannelLink,
            title: newChannelLink 
        });
        await SaveTgChannel(channel);
        newChannelLink = "";
        openAddChanel = false;
        await refreshAll();
    }

    async function handleSaveTemplate() {
        if (!newTemplateName) return;
        
        // При сохранении шаблона сохраняем сырой текст. 
        // Если пользователь хочет сохранить шаблон с переменными, 
        // он должен написать {{article_name}} вручную в поле ввода.
        const tmpl = new database.TgTemplate({
            name: newTemplateName,
            content: postText 
        });
        await SaveTgTemplate(tmpl);
        newTemplateName = "";
        openAddTemplate = false;
        await refreshAll();
    }

    async function handleSend() {
        if (!selectedBotId || !selectedChannelId || !postText) {
            alert("Заполните бота, канал и текст");
            return;
        }

        const bot = bots.find(b => b.id === selectedBotId);
        if (!bot) return;

        // Логика расчета времени для отложенной отправки
        let scheduleUnix = 0;

        if (postDate && postTime) {
            try {
                // Создаем дату из строк. Важно: DateField обычно возвращает YYYY-MM-DD
                const dateTimeString = `${postDate}T${postTime}`;
                const scheduledDate = new Date(dateTimeString);
                
                // Проверка на валидность даты
                if (isNaN(scheduledDate.getTime())) {
                    alert("Некорректная дата или время");
                    return;
                }

                // Получаем Unix Timestamp в секундах
                scheduleUnix = Math.floor(scheduledDate.getTime() / 1000);

                // Проверка: дата должна быть в будущем
                const nowUnix = Math.floor(Date.now() / 1000);
                if (scheduleUnix <= nowUnix) {
                    alert("Дата отложенной отправки должна быть в будущем!");
                    return;
                }

            } catch (e) {
                console.error("Ошибка парсинга даты:", e);
                alert("Ошибка обработки даты");
                return;
            }
        } else if (postDate || postTime) {
            alert("Для отложенной отправки заполните и дату, и время.");
            return;
        }
        
        try {
            // Передаем scheduleUnix 4-м параметром (в Go это аргумент int64)
            await SendToTelegram(bot.token, selectedChannelId, postText, scheduleUnix); 
            
            if (scheduleUnix > 0) {
                alert(`Успешно запланировано на ${postDate} ${postTime}!`);
            } else {
                alert("Успешно отправлено!");
            }
        } catch (e) {
            alert("Ошибка отправки: " + e);
        }
    }
</script>

<div class="telegram-upload">
    <Dialog headline="Новый бот" bind:open={openAddBot} style="margin: auto;">
        <TextField bind:value={newBotToken} label="Токен бота" />
        {#snippet buttons()}
            <Button variant="text" onclick={() => openAddBot = false}>Отмена</Button>
            <Button variant="tonal" onclick={handleSaveBot}>Oк</Button>
        {/snippet}
    </Dialog>

    <Dialog
        headline="Новый канал"
        bind:open={openAddChanel}
        style="margin: auto;"
    >
        <div style="margin-bottom: 10px; font-size: 0.8em; opacity: 0.7">
            Канал будет привязан к текущему выбранному боту.
        </div>
        <TextField bind:value={newChannelLink} label="ID канала или ссылка (@chan)" />
        {#snippet buttons()}
            <Button variant="text" onclick={() => openAddChanel = false}>Отмена</Button>
            <Button variant="tonal" onclick={handleSaveChannel}>Oк</Button>
        {/snippet}
    </Dialog>

    <Dialog
        headline="Новый шаблон"
        bind:open={openAddTemplate}
        style="margin: auto;"
    >
        <TextField bind:value={newTemplateName} label="Название шаблона" />
        <div style="margin-top: 10px; font-size: 0.8em; opacity: 0.7">
            Текущий текст из поля ввода будет сохранен.
        </div>
        {#snippet buttons()}
            <Button variant="text" onclick={() => openAddTemplate = false}>Отмена</Button>
            <Button variant="tonal" onclick={handleSaveTemplate}>Oк</Button>
        {/snippet}
    </Dialog>

    <Card variant="filled">
        <div class="select-group">
            <div class="bot-select">
                <div class="select">
                    <SelectOutlined
                        label="Бот"
                        options={botOptions}
                        bind:value={selectedBotId}
                    />
                </div>
                <Button
                    variant="filled"
                    square
                    size="m"
                    onclick={() => (openAddBot = true)}
                >
                    <Icon icon={addIcon} />
                </Button>
            </div>
            <div class="bot-select">
                <div class="select">
                    <SelectOutlined
                        label="Канал"
                        options={filteredChannelOptions}
                        bind:value={selectedChannelId}
                        disabled={!selectedBotId} 
                    />
                </div>
                <Button
                    variant="filled"
                    square
                    size="m"
                    onclick={() => (openAddChanel = true)}
                    disabled={!selectedBotId}
                >
                    <Icon icon={addIcon} />
                </Button>
            </div>
        </div>
    </Card>
    <br />

    <Card variant="filled">
        <div class="edit-post">
            <div class="bot-select">
                <div class="select">
                    <SelectOutlined
                        label="Шаблон"
                        options={templateOptions}
                        bind:value={selectedTemplateId}
                        onchange={onTemplateSelect}
                    />
                </div>
                <Button 
                    variant="filled" 
                    square 
                    size="m"
                    onclick={() => openAddTemplate = true}
                >
                    <Icon icon={addIcon} />
                </Button>
            </div>
            <TextFieldOutlinedMultiline 
                bind:value={postText} 
                label="Текст поста" 
            />
        </div>
    </Card>
    <br />

    <Card variant="filled">
        <div class="date-select">
            <DateField label="Дата отправки" bind:value={postDate} />
            <TextField label="Время отправки" type="time" bind:value={postTime} />
            
            <Button variant="filled" onclick={handleSend}>
                <Icon icon={sendIcon} />
                {#if postDate && postTime}
                    Отложить
                {:else}
                    Отправить
                {/if}
            </Button>
        </div>
    </Card>
</div>

<style>
    .select-group {
        display: flex;
        justify-content: space-between;
        gap: 16px;
    }
    .bot-select {
        flex-grow: 1;
        display: flex;
        flex-direction: row;
        justify-content: space-between;
        gap: 10px;
    }
    .select {
        flex-grow: 1;
    }
    .edit-post {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }
    .date-select {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }
</style>
