<script>
    import { TextField, Button, Icon, Dialog } from "m3-svelte";
    import { titlesStore } from "../stores/titles.svelte";

    // Icons
    import iconFolder from "@ktibow/iconset-material-symbols/folder-open-outline";
    import iconImage from "@ktibow/iconset-material-symbols/image-outline";
    import iconAdd from "@ktibow/iconset-material-symbols/add";

    let {
        chapterTitle = $bindable(""),
        isProcessing = false,
        onSelectFolder,
        onSelectFiles,
    } = $props();

    let showNewTitleDialog = $state(false);
    let newTitleName = $state("");
    let newTitleFolder = $state("");

    function handleCreateTitle() {
        if (!newTitleName.trim()) return;
        titlesStore.createTitleAction(newTitleName, newTitleFolder);
        newTitleName = "";
        newTitleFolder = "";
        showNewTitleDialog = false;
    }

    async function handleSelectTitleFolder() {
        try {
            const result = await onSelectFolder?.(true); // Helper hack or separate method needed?
            // Wait, onSelectFolder in props calls appState.selectFolderAction which targets main content.
            // We need a direct way to open folder dialog without triggering appState logic.
            // Let's use Wails Runtime directly here or add a helper in props/store.
            // Actually, we can just call OpenFolderDialog from Wails JS directly here.
        } catch (e) {
            console.error(e);
        }
    }
</script>

<script module>
    import { OpenFolderDialog } from "../../wailsjs/go/main/App";
    async function pickFolder() {
       try {
            const res = await OpenFolderDialog();
            return res.path;
       } catch(e) {
           return "";
       }
    }
</script>

<header class="header-container">
    <div class="input-wrapper">
        <TextField
            bind:value={chapterTitle}
            label="Название главы"
            disabled={isProcessing}
            type="text"
        />
    </div>

    <div class="title-select-wrapper">
        <select
            class="native-select"
            bind:value={titlesStore.selectedTitleId}
            disabled={isProcessing}
        >
            <option value={0}>Без тайтла</option>
            {#each titlesStore.titles as title}
                <option value={title.id}>{title.name}</option>
            {/each}
        </select>
        <Button
            variant="tonal"
            onclick={() => (showNewTitleDialog = true)}
            disabled={isProcessing}
        >
            <Icon icon={iconAdd} />
        </Button>
    </div>

    <Button
        variant="filled"
        onclick={() => onSelectFolder?.()}
        disabled={isProcessing}
        iconType="left"
    >
        <Icon icon={iconFolder} />
        Папка
    </Button>

    <Button
        variant="filled"
        onclick={() => onSelectFiles?.()}
        disabled={isProcessing}
    >
        <Icon icon={iconImage} />
        Файлы
    </Button>
</header>

<Dialog bind:open={showNewTitleDialog} headline="Новый тайтл">
    <div class="dialog-content">
        <TextField
            bind:value={newTitleName}
            label="Название тайтла"
            type="text"
        />
        <div class="folder-select-row" style="margin-top: 10px; display: flex; align-items: center; gap: 8px;">
            <TextField
                 value={newTitleFolder}
                 label="Папка (необязательно)"
                 type="text"
                 readonly
                 style="flex-grow: 1;"
            />
            <Button variant="tonal" onclick={async () => {
                const path = await pickFolder();
                if (path) newTitleFolder = path;
            }}>
                <Icon icon={iconFolder} />
            </Button>
        </div>
    </div>
    {#snippet buttons()}
        <Button variant="text" onclick={() => (showNewTitleDialog = false)}
            >Отмена</Button
        >
        <Button variant="text" onclick={handleCreateTitle}>Создать</Button>
    {/snippet}
</Dialog>

<style>
    .header-container {
        display: flex;
        align-items: center;
        width: 100%;
        gap: 10px;
    }
    .input-wrapper {
        flex-grow: 1;
        display: flex;
        flex-direction: column;
    }
    .title-select-wrapper {
        display: flex;
        align-items: center;
        gap: 5px;
    }
    .native-select {
        height: 56px;
        border-radius: 4px 4px 0 0;
        background-color: var(--m3c-surface-container-highest);
        color: var(--m3c-on-surface);
        border: none;
        border-bottom: 1px solid var(--m3c-outline);
        padding: 0 16px;
        font-size: 16px;
        outline: none;
        appearance: none; /* Remove default arrow in some browsers */
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' height='24' viewBox='0 -960 960 960' width='24' fill='%23444746'%3E%3Cpath d='M480-345 240-585l56-56 184 184 184-184 56 56-240 240Z'/%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 8px center;
        padding-right: 40px;
        min-width: 150px;
    }
    .native-select:focus {
        border-bottom: 2px solid var(--m3c-primary);
    }
    .dialog-content {
        padding: 10px 0;
        width: 100%;
    }
</style>
