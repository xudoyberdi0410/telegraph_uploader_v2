<script>
    import { TextField, Button, Icon } from "m3-svelte";

    // Иконки
    import iconFolder from "@ktibow/iconset-material-symbols/folder-open-outline";
    import iconImage from "@ktibow/iconset-material-symbols/image-outline";

    let {
        chapterTitle = $bindable(""),
        isProcessing = false,
        onSelectFolder,
        onSelectFiles
    } = $props()
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
</style>
