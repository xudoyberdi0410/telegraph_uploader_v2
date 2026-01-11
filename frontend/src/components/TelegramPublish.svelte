<script>
    import { appState } from "../stores/appStore.svelte";
    import {
        SearchChannels,
        GetTemplates,
        GetTitleByID,
        PublishPost,
        CreateTemplate,
        AddTitleVariable,
    } from "../../wailsjs/go/main/App";
    import { onMount, tick } from "svelte";
    import {
        Button,
        TextFieldOutlined,
        SelectOutlined,
        Dialog,
        Icon,
    } from "m3-svelte";

    import iconFormatBold from "@ktibow/iconset-material-symbols/format-bold";
    import iconFormatItalic from "@ktibow/iconset-material-symbols/format-italic";
    import iconFormatUnderlined from "@ktibow/iconset-material-symbols/format-underlined";
    import iconStrikethroughS from "@ktibow/iconset-material-symbols/strikethrough-s";
    import iconLink from "@ktibow/iconset-material-symbols/link";
    import iconFormatListBulleted from "@ktibow/iconset-material-symbols/format-list-bulleted";
    import iconFormatListNumbered from "@ktibow/iconset-material-symbols/format-list-numbered";
    import iconUndo from "@ktibow/iconset-material-symbols/undo";
    import iconRedo from "@ktibow/iconset-material-symbols/redo";
    import iconSearch from "@ktibow/iconset-material-symbols/search";
    import iconContentCopy from "@ktibow/iconset-material-symbols/content-copy";
    import iconAdd from "@ktibow/iconset-material-symbols/add";

    let { historyId = 0, articleTitle = "", titleId = 0 } = $props();

    let channels = $state([]);
    let templates = $state([]);
    let customVariables = $state([]);

    // Form State
    let selectedChannelId = $state("");
    let selectedChannelTitle = $state(""); // For display in search input
    let selectedTitleId = $state("0");
    let selectedTemplateId = $state("");
    let message = $state("");
    // Schedule
    let scheduleDateDate = $state("");
    let scheduleDateTime = $state("");

    let isSubmitting = $state(false);

    // UI State
    let showAddVariableDialog = $state(false);
    let showSaveTemplateDialog = $state(false);
    let snackbarMessage = $state("");
    let showSnackbar = $state(false);
    let showChannelDropdown = $state(false);

    // Dialog Inputs
    let newVarKey = $state("");
    let newVarValue = $state("");
    let newTemplateName = $state("");
    let searchTerm = $state("");

    let editorTextarea;

    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    async function handleSearchInput(e) {
        searchTerm = e.target.value;
        if (searchTerm.length >= 2) {
            try {
                channels = (await SearchChannels(searchTerm)) || [];
                showChannelDropdown = true;
            } catch (error) {
                console.error("Search failed:", error);
                channels = [];
            }
        } else {
            // channels = [];
        }
    }

    function selectChannel(channel) {
        selectedChannelId = String(channel.id);
        selectedChannelTitle = channel.title;
        searchTerm = channel.title;
        showChannelDropdown = false;
        selectedChannelObj = channel;

        // Save to settings
        appState.settings.last_channel_id = String(channel.id);
        appState.settings.last_channel_hash = String(channel.access_hash);
        appState.settings.last_channel_title = channel.title;
        appState.triggerAutoSave();
    }

    async function loadTemplates() {
        templates = (await GetTemplates()) || [];
    }

    onMount(async () => {
        try {
            if (!historyId || historyId === 0) {
                // Prevent access without context
                appState.currentPage = "home";
                return;
            }

            await loadTemplates();

            if (titleId) selectedTitleId = String(titleId);
            else if (appState.currentTitleId)
                selectedTitleId = String(appState.currentTitleId);
            else if (appState.selectedTitleId)
                selectedTitleId = String(appState.selectedTitleId);

            if (selectedTitleId && selectedTitleId !== "0")
                await loadTitleVariables(Number(selectedTitleId));

            // Click outside to close dropdown
            document.addEventListener("click", (e) => {
                const searchWrap = document.querySelector(
                    ".channel-search-wrapper",
                );
                if (searchWrap && !searchWrap.contains(e.target)) {
                    showChannelDropdown = false;
                }
            });
        } catch (e) {
            console.error("Failed to load initial data:", e);
        }
    });

    async function loadTitleVariables(titleId) {
        const id = Number(titleId);
        if (!id) {
            customVariables = [];
            return;
        }
        try {
            const title = await GetTitleByID(id);
            if (title && title.variables) {
                customVariables = title.variables.filter(
                    (v) => v.key && v.value,
                );
            } else {
                customVariables = [];
            }
        } catch (e) {
            console.error("Failed to load title variables", e);
            customVariables = [];
        }
    }

    async function handleTitleChange() {
        await loadTitleVariables(selectedTitleId);
    }

    async function handleAddVariable() {
        if (!newVarKey.trim() || !newVarValue.trim()) {
            showToast("Заполните ключ и значение!");
            return;
        }
        try {
            await AddTitleVariable(
                Number(selectedTitleId),
                newVarKey,
                newVarValue,
            );
            await loadTitleVariables(Number(selectedTitleId));
            newVarKey = "";
            newVarValue = "";
            showAddVariableDialog = false;
        } catch (e) {
            showToast("Ошибка: " + e);
        }
    }

    function selectTemplate() {
        const tmpl = templates.find((t) => t.id == selectedTemplateId);
        if (tmpl) {
            message = tmpl.content;
        }
    }

    async function handleSaveTemplate() {
        if (!newTemplateName.trim()) {
            showToast("Введите название шаблона!");
            return;
        }
        try {
            await CreateTemplate(newTemplateName, message);
            await loadTemplates();
            newTemplateName = "";
            showSaveTemplateDialog = false;
            showToast("Шаблон сохранен!");
        } catch (e) {
            showToast("Ошибка: " + e);
        }
    }

    function insertText(text) {
        if (!editorTextarea) {
            message += text;
            return;
        }

        const start = editorTextarea.selectionStart;
        const end = editorTextarea.selectionEnd;
        const before = message.substring(0, start);
        const after = message.substring(end);

        message = before + text + after;

        // Restore focus and cursor position (basic)
        tick().then(() => {
            editorTextarea.focus();
            const newPos = start + text.length;
            editorTextarea.setSelectionRange(newPos, newPos);
        });
    }

    function formatText(type) {
        if (!editorTextarea) return;

        const start = editorTextarea.selectionStart;
        const end = editorTextarea.selectionEnd;
        const selected = message.substring(start, end);
        const before = message.substring(0, start);
        const after = message.substring(end);

        let formatted = "";

        switch (type) {
            case "bold":
                formatted = `<b>${selected}</b>`;
                break;
            case "italic":
                formatted = `<i>${selected}</i>`;
                break;
            case "underline":
                formatted = `<u>${selected}</u>`;
                break;
            case "strike":
                formatted = `<s>${selected}</s>`;
                break;
            case "link":
                formatted = `<a href="url">${selected}</a>`;
                break;
            default:
                formatted = selected;
        }

        message = before + formatted + after;
        tick().then(() => {
            editorTextarea.focus();
            const newPos = start + formatted.length;
            editorTextarea.setSelectionRange(newPos, newPos);
        });
    }

    async function handlePublish() {
        if (!selectedChannelId) {
            showToast("Выберите канал!");
            return;
        }
        if (!historyId) {
            showToast("Ошибка: Не выбрана статья (History ID missing).");
            return;
        }

        isSubmitting = true;
        try {
            let dateStr = "";
            if (scheduleDateDate && scheduleDateTime) {
                dateStr = new Date(
                    `${scheduleDateDate}T${scheduleDateTime}`,
                ).toISOString();
            } else if (scheduleDateDate || scheduleDateTime) {
                showToast("Для отложенной отправки укажите дату и время");
                isSubmitting = false;
                return;
            } else {
                dateStr = new Date().toISOString();
            }

            if (!selectedChannelObj) {
                // Try to find in current list
                selectedChannelObj = channels.find(
                    (c) => String(c.id) === selectedChannelId,
                );
            }

            if (!selectedChannelObj)
                throw new Error(
                    "Channel information missing (access_hash). Search again and select channel.",
                );

            await PublishPost(
                historyId,
                selectedChannelObj.id,
                selectedChannelObj.access_hash,
                message,
                dateStr,
            );

            showToast("Опубликовано успешно!");
        } catch (e) {
            showToast("Ошибка: " + e.message);
        } finally {
            isSubmitting = false;
        }
    }

    let selectedChannelObj = $state(null);
    $effect(() => {
        if (selectedChannelId && channels.length > 0) {
            const found = channels.find(
                (c) => String(c.id) === selectedChannelId,
            );
            if (found) selectedChannelObj = found;
        }
    });

    // Load Last Selected Channel
    $effect(() => {
        if (
            !selectedChannelId &&
            appState.settings.last_channel_id &&
            appState.settings.last_channel_id !== "0"
        ) {
            const lastId = appState.settings.last_channel_id;
            const lastTitle =
                appState.settings.last_channel_title || "Unknown Channel";
            const lastHash = appState.settings.last_channel_hash;

            selectedChannelId = lastId;
            selectedChannelTitle = lastTitle;
            searchTerm = lastTitle;

            // We construct the object so we can publish without re-searching
            selectedChannelObj = {
                id: lastId,
                access_hash: lastHash,
                title: lastTitle,
            };
        }
    });

    function showToast(msg) {
        snackbarMessage = msg;
        showSnackbar = true;
        setTimeout(() => (showSnackbar = false), 4000);
    }

    // Derived
    let titleOptions = $derived([
        { text: "Без тайтла", value: "0" },
        ...appState.titles.map((t) => ({ text: t.name, value: String(t.id) })),
    ]);
    let templateOptions = $derived([
        { text: "Выберите шаблон", value: "" },
        ...templates.map((t) => ({ text: t.name, value: String(t.id) })),
    ]);
</script>

<div class="main-container">
    <div class="header">
        <h1>
            Публикация{#if articleTitle}<span class="article-title">
                    - {articleTitle}</span
                >{/if}
        </h1>
    </div>

    <div class="top-row">
        <!-- Settings Bar -->
        <div class="settings-group">
            <div class="control-item big-search">
                <div class="channel-search-wrapper">
                    <div class="search-input-box">
                        <Icon icon={iconSearch} class="search-icon" />
                        <input
                            type="text"
                            placeholder="Поиск канала..."
                            bind:value={searchTerm}
                            oninput={handleSearchInput}
                            onfocus={() => {
                                if (searchTerm.length > 0)
                                    showChannelDropdown = true;
                            }}
                        />
                    </div>
                    {#if showChannelDropdown && channels.length > 0}
                        <div class="dropdown-list">
                            {#each channels as channel}
                                <button
                                    class="dropdown-item"
                                    onclick={() => selectChannel(channel)}
                                >
                                    <div class="channel-title">
                                        {channel.title}
                                    </div>
                                    {#if channel.username}<div
                                            class="channel-username"
                                        >
                                            @{channel.username}
                                        </div>{/if}
                                </button>
                            {/each}
                        </div>
                    {:else if showChannelDropdown && searchTerm.length >= 2}
                        <div class="dropdown-list">
                            <div class="dropdown-item empty">
                                Ничего не найдено
                            </div>
                        </div>
                    {/if}
                </div>
            </div>

            <div class="control-item">
                <SelectOutlined
                    label="Тайтл (Context)"
                    bind:value={selectedTitleId}
                    options={titleOptions}
                    onchange={handleTitleChange}
                />
            </div>

            <div class="control-item">
                <SelectOutlined
                    label="Выберите шаблон"
                    bind:value={selectedTemplateId}
                    options={templateOptions}
                    onchange={selectTemplate}
                />
            </div>

            <button class="icon-btn-outline" title="Copy from template">
                <Icon icon={iconContentCopy} />
            </button>

            <button
                class="btn-outline"
                onclick={() => (showSaveTemplateDialog = true)}
            >
                Сохранить шаблон
            </button>
        </div>

        <!-- Variables Row (Just below settings or inline? Let's inline or next row) -->
        <div class="variables-bar">
            <strong>Переменные:</strong>
            <button class="var-chip" onclick={() => insertText("{{Link}}")}
                >Link</button
            >
            <button class="var-chip" onclick={() => insertText("{{Title}}")}
                >Title</button
            >
            {#each customVariables as v}
                <button
                    class="var-chip"
                    onclick={() => insertText(`{{${v.key}}}`)}>{v.key}</button
                >
            {/each}
            {#if selectedTitleId && selectedTitleId !== "0"}
                <button
                    class="var-add-btn"
                    onclick={() => (showAddVariableDialog = true)}
                >
                    <Icon icon={iconAdd} size="1em" />
                </button>
            {/if}
        </div>
    </div>

    <div class="editor-area">
        <div class="editor-wrapper">
            <div class="editor-toolbar">
                <button
                    class="toolbar-btn"
                    onclick={() => formatText("bold")}
                    title="Bold"
                >
                    <Icon icon={iconFormatBold} size="1.2em" />
                </button>
                <button
                    class="toolbar-btn"
                    onclick={() => formatText("italic")}
                    title="Italic"
                >
                    <Icon icon={iconFormatItalic} size="1.2em" />
                </button>
                <button
                    class="toolbar-btn"
                    onclick={() => formatText("underline")}
                    title="Underline"
                >
                    <Icon icon={iconFormatUnderlined} size="1.2em" />
                </button>
                <button
                    class="toolbar-btn"
                    onclick={() => formatText("strike")}
                    title="Strikethrough"
                >
                    <Icon icon={iconStrikethroughS} size="1.2em" />
                </button>
                <div class="toolbar-divider"></div>
                <button
                    class="toolbar-btn"
                    onclick={() => formatText("link")}
                    title="Link"
                >
                    <Icon icon={iconLink} size="1.2em" />
                </button>
                <div class="toolbar-spacer"></div>
                <button class="toolbar-btn" title="Undo">
                    <Icon icon={iconUndo} size="1.2em" />
                </button>
                <button class="toolbar-btn" title="Redo">
                    <Icon icon={iconRedo} size="1.2em" />
                </button>
            </div>
            <textarea
                bind:this={editorTextarea}
                bind:value={message}
                placeholder="Текст сообщения"
                class="editor-textarea"
            ></textarea>
        </div>
    </div>

    <div class="bottom-bar">
        <!-- Date Time aligned left -->
        <div class="schedule-group">
            <div class="dt-input">
                <input type="date" bind:value={scheduleDateDate} />
            </div>
            <div class="dt-input">
                <input type="time" bind:value={scheduleDateTime} />
            </div>
        </div>

        <div class="spacer"></div>

        <!-- Action Buttons aligned right -->
        <div class="action-buttons">
            <button
                class="btn-secondary"
                onclick={() => showToast("Черновик сохранен (mock)")}
            >
                Сохранить черновик
            </button>
            <button
                class="btn-primary"
                onclick={handlePublish}
                disabled={isSubmitting}
            >
                {isSubmitting ? "Отправка..." : "Опубликовать"}
            </button>
        </div>
    </div>
</div>

<!-- Dialogs -->
<Dialog
    bind:open={showAddVariableDialog}
    headline="Новая переменная"
    style="margin: auto"
>
    <div class="dialog-form">
        <TextFieldOutlined label="Ключ (без скобок)" bind:value={newVarKey} />
        <TextFieldOutlined label="Значение" bind:value={newVarValue} />
    </div>
    {#snippet buttons()}
        <Button variant="text" onclick={() => (showAddVariableDialog = false)}
            >Отмена</Button
        >
        <Button variant="text" onclick={handleAddVariable}>Добавить</Button>
    {/snippet}
</Dialog>

<Dialog
    bind:open={showSaveTemplateDialog}
    headline="Сохранить шаблон"
    style="margin: auto"
>
    <div class="dialog-form">
        <TextFieldOutlined
            label="Название шаблона"
            bind:value={newTemplateName}
        />
    </div>
    {#snippet buttons()}
        <Button variant="text" onclick={() => (showSaveTemplateDialog = false)}
            >Отмена</Button
        >
        <Button variant="text" onclick={handleSaveTemplate}>Сохранить</Button>
    {/snippet}
</Dialog>

<!-- Snackbar -->
{#if showSnackbar}
    <div class="snackbar-wrapper">
        <div class="custom-snackbar">{snackbarMessage}</div>
    </div>
{/if}

<style>
    .main-container {
        padding: 16px;
        height: 100%;
        box-sizing: border-box;
        background-color: var(--m3-sys-color-background, #121212);
        color: var(--m3-sys-color-on-background, #fff);
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .header h1 {
        font-family: "Roboto", sans-serif;
        font-size: 24px;
        font-weight: 500;
        margin: 0 0 8px 0;
        color: var(--m3-sys-color-on-background, #fff);
    }

    .article-title {
        color: var(--m3-sys-color-primary, #a8c7fa);
        font-weight: 400;
        font-size: 0.9em;
    }

    /* Top Row Settings */
    .top-row {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .settings-group {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap; /* Allow wrapping securely */
    }

    .control-item {
        min-width: 200px;
    }
    .big-search {
        flex: 1; /* Search bar takes available space or at least 300px */
        min-width: 300px;
    }

    /* Variables Bar */
    .variables-bar {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
        font-size: 14px;
        color: #aaa;
    }

    .var-chip {
        background: rgba(255, 255, 255, 0.08); /* Slightly lighter */
        border: 1px solid #444;
        border-radius: 6px;
        padding: 4px 10px;
        color: #ddd;
        font-size: 13px;
        cursor: pointer;
        display: flex;
        align-items: center;
        transition: background 0.2s;
    }
    .var-chip:hover {
        background: rgba(255, 255, 255, 0.15);
        color: #fff;
    }
    .var-add-btn {
        background: transparent;
        border: 1px dashed #666;
        border-radius: 50%;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #aaa;
        cursor: pointer;
    }
    .var-add-btn:hover {
        border-color: #aaa;
        color: #fff;
    }

    /* Channel Search */
    .channel-search-wrapper {
        position: relative;
        width: 100%;
        z-index: 50; /* Above other inputs */
    }
    .search-input-box {
        display: flex;
        align-items: center;
        background-color: #2c2c2c;
        border: 1px solid #555;
        border-radius: 8px; /* M3 */
        padding: 0 12px;
        height: 56px; /* Match standard m3 height roughly */
        transition: border-color 0.2s;
    }
    .search-input-box:focus-within {
        border-color: var(--m3-sys-color-primary, #a8c7fa);
        border-width: 2px;
        padding: 0 11px;
    }

    .search-input-box input {
        background: transparent;
        border: none;
        color: #fff;
        flex: 1;
        font-size: 16px;
        outline: none;
        height: 100%;
    }
    .dropdown-list {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        margin-top: 4px;
        background-color: #2c2c2c;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
        max-height: 250px;
        overflow-y: auto;
        border: 1px solid #444;
    }
    .dropdown-item {
        width: 100%;
        text-align: left;
        background: transparent;
        border: none;
        padding: 12px 16px;
        color: #eee;
        cursor: pointer;
        border-bottom: 1px solid #383838;
    }
    .dropdown-item:last-child {
        border-bottom: none;
    }
    .dropdown-item:hover {
        background-color: #3d3d3d;
    }
    .channel-title {
        font-weight: 500;
        font-size: 15px;
    }
    .channel-username {
        font-size: 13px;
        color: #999;
    }
    .dropdown-item.empty {
        color: #777;
        font-style: italic;
        cursor: default;
    }

    /* Buttons reuse */
    .btn-outline {
        background: transparent;
        border: 1px solid #555;
        color: #ddd;
        padding: 0 16px;
        height: 56px;
        border-radius: 4px; /* Match m3 radius mostly 4px for inputs */
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        white-space: nowrap;
    }
    .btn-outline:hover {
        border-color: #888;
        color: #fff;
    }

    .icon-btn-outline {
        background: transparent;
        border: 1px solid #555;
        border-radius: 4px;
        width: 56px;
        height: 56px;
        color: #ccc;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    .icon-btn-outline:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    /* Editor Area */
    .editor-area {
        flex: 1; /* Go full height */
        display: flex;
        flex-direction: column;
        min-height: 200px;
    }
    .editor-wrapper {
        border: 1px solid #444;
        border-radius: 12px;
        display: flex;
        flex-direction: column;
        flex: 1; /* Stretch */
        background-color: #1e1e1e;
        overflow: hidden;
        transition: border-color 0.2s;
    }
    .editor-wrapper:focus-within {
        border-color: var(--m3-sys-color-primary, #a8c7fa);
    }

    .editor-toolbar {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 8px 12px;
        background-color: #252525;
        border-bottom: 1px solid #444;
    }
    .toolbar-btn {
        background: transparent;
        border: none;
        color: #aaa;
        padding: 6px;
        border-radius: 4px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .toolbar-btn:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
    }
    .toolbar-divider {
        width: 1px;
        height: 20px;
        background: #555;
        margin: 0 4px;
    }
    .toolbar-spacer {
        flex: 1;
    }

    .editor-textarea {
        flex: 1;
        background: transparent;
        border: none;
        color: var(--m3-sys-color-on-surface, #e3e3e3);
        padding: 24px; /* More padding for full page feel */
        font-family: inherit;
        font-size: 16px;
        resize: none;
        outline: none;
        line-height: 1.6;
    }

    /* Bottom Bar */
    .bottom-bar {
        display: flex;
        align-items: center;
        gap: 20px;
        margin-top: auto; /* Push to bottom if container grows */
        background-color: #1a1a1a; /* Distinct bar bg? or just transp? */
        padding: 16px 24px;
        border-radius: 12px;
        border: 1px solid #333;
    }
    .schedule-group {
        display: flex;
        gap: 12px;
    }
    .dt-input {
        background: #2c2c2c;
        border: 1px solid #555;
        border-radius: 8px;
        height: 48px;
        display: flex;
        align-items: center;
        padding: 0 12px;
    }
    .dt-input input {
        background: transparent;
        border: none;
        color: #fff;
        font-family: inherit;
        font-size: 14px;
        outline: none;
        color-scheme: dark;
    }

    .spacer {
        flex: 1;
    }

    .action-buttons {
        display: flex;
        gap: 12px;
    }
    .btn-primary,
    .btn-secondary {
        padding: 0 24px;
        height: 48px;
        border-radius: 24px; /* Pill shape for primary actions */
        font-size: 15px;
        font-weight: 500;
        cursor: pointer;
        border: none;
        transition: filter 0.2s;
        white-space: nowrap;
    }
    .btn-primary {
        background-color: var(--m3-sys-color-primary, #64b5f6);
        color: var(--m3-sys-color-on-primary, #003258);
    }
    .btn-primary:hover {
        filter: brightness(1.1);
    }
    .btn-primary:active {
        filter: brightness(0.9);
    }
    .btn-primary:disabled {
        background: #333;
        color: #777;
        cursor: not-allowed;
    }

    .btn-secondary {
        background: transparent;
        border: 1px solid #555;
        color: #ddd;
    }
    .btn-secondary:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .dialog-form {
        display: flex;
        flex-direction: column;
        gap: 16px;
        padding-top: 8px;
        min-width: 300px;
    }

    .snackbar-wrapper {
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 2000;
    }
    .custom-snackbar {
        background-color: #333;
        color: #fff;
        padding: 14px 16px;
        border-radius: 4px;
        box-shadow: 0 3px 5px rgba(0, 0, 0, 0.3);
    }

    @media (max-width: 900px) {
        .settings-group {
            flex-direction: column;
            align-items: stretch;
        }
        .control-item,
        .big-search {
            width: 100%;
            min-width: 0;
        }
        .bottom-bar {
            flex-direction: column;
            align-items: stretch;
            height: auto;
        }
        .schedule-group {
            width: 100%;
        }
        .dt-input {
            flex: 1;
        }
        .action-buttons {
            width: 100%;
        }
        .btn-primary,
        .btn-secondary {
            flex: 1;
        }
    }
</style>
