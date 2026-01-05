<script>
    import { onMount } from "svelte";
    import {
        GetTgBots,
        GetTgChannels,
        GetTgTemplates,
        SendToTelegram,
    } from "../../wailsjs/go/main/App";
    import { chapterTitle, finalUrl } from "../stores/appStore";

    export let close;

    let bots = [],
        channels = [],
        templates = [];
    let selectedBot = "",
        selectedChannel = "",
        selectedTemplate = "";
    let postText = "";
    let scheduleDate = "";

    onMount(async () => {
        bots = await GetTgBots();
        channels = await GetTgChannels();
        templates = await GetTgTemplates();
    });

    function applyTemplate() {
        const t = templates.find((x) => x.id == selectedTemplate);
        if (t) {
            postText = t.content
                .replace("{{title}}", $chapterTitle)
                .replace("{{url}}", $finalUrl);
        }
    }

    async function send() {
        const unix = scheduleDate ? new Date(scheduleDate).getTime() / 1000 : 0;
        const res = await SendToTelegram(
            selectedBot,
            selectedChannel,
            postText,
            unix,
        );
        alert(res);
        if (res.includes("Успешно")) close();
    }
</script>

<div class="modal-overlay">
    <div class="modal">
        <h2>Отправить в Telegram</h2>

        <label>Бот:</label>
        <select bind:value={selectedBot}>
            {#each bots as bot}<option value={bot.token}>{bot.name}</option
                >{/each}
        </select>

        <label>Канал:</label>
        <select bind:value={selectedChannel}>
            {#each channels as ch}<option value={ch.channel_id}
                    >{ch.title}</option
                >{/each}
        </select>

        <label>Шаблон:</label>
        <select bind:value={selectedTemplate} on:change={applyTemplate}>
            {#each templates as t}<option value={t.id}>{t.name}</option>{/each}
        </select>

        <label>Текст поста:</label>
        <textarea bind:value={postText} rows="5"></textarea>

        <label>Дата отложки (пусто = сейчас):</label>
        <input type="datetime-local" bind:value={scheduleDate} />

        <div class="actions">
            <button on:click={close}>Отмена</button>
            <button class="primary" on:click={send}>Запланировать</button>
        </div>
    </div>
</div>

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .modal {
        background: #2a2a2a;
        padding: 20px;
        border-radius: 8px;
        width: 400px;
        display: flex;
        flex-direction: column;
        gap: 10px;
        color: white;
    }
    textarea {
        background: #1a1a1a;
        color: white;
        border: 1px solid #444;
        padding: 5px;
    }
    select,
    input {
        padding: 8px;
        background: #333;
        color: white;
        border: 1px solid #444;
    }
    .primary {
        background: #0088cc;
        color: white;
        border: none;
        padding: 10px;
        cursor: pointer;
    }
</style>
