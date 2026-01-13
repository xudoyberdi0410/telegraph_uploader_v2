<script>
    import { onMount } from "svelte";
    import { ListFiles, DeleteFiles } from "../../wailsjs/go/main/App";
    import { Button, Icon } from "m3-svelte";
    import iconDelete from "@ktibow/iconset-material-symbols/delete-outline";
    import iconExpand from "@ktibow/iconset-material-symbols/expand-more";
    import iconCollapse from "@ktibow/iconset-material-symbols/expand-less";

    // Optimization: Pre-create formatter to avoid recreation in loops
    const dateFormatter = new Intl.DateTimeFormat('default', {
        dateStyle: 'medium',
        timeStyle: 'short'
    });

    let allFiles = $state([]);
    let timeThreshold = $state(1);
    // Separate slider value for immediate visual feedback without costly recalc
    let sliderValue = $state(1);
    let isLoading = $state(false);
    
    // Derived state logic for grouping
    let groupedChapters = $derived.by(() => {
        if (!allFiles || allFiles.length === 0) return [];

        const sorted = [...allFiles].sort((a, b) => b.last_modified - a.last_modified);
        const groups = [];
        let currentGroup = [];

        // Helper to create group object
        const createGroup = (files) => {
             const first = files[0];
             // Optimization: Deterministic ID based on content to prevent re-renders
             // Using name + timestamp ensures ID is stable as long as data is same
             const id = `g-${first.name}-${first.last_modified}`;
             return {
                 id,
                 files,
                 // Optimization: Date formatting using cached formatter
                 date: dateFormatter.format(new Date(first.last_modified * 1000))
             };
        };

        for (const file of sorted) {
            if (currentGroup.length === 0) {
                currentGroup.push(file);
                continue;
            }

            const firstFile = currentGroup[0];
            const diffMinutes = Math.abs(file.last_modified - firstFile.last_modified) / 60;

            if (diffMinutes <= timeThreshold) {
                currentGroup.push(file);
            } else {
                groups.push(createGroup(currentGroup));
                currentGroup = [file];
            }
        }

        if (currentGroup.length > 0) {
            groups.push(createGroup(currentGroup));
        }

        return groups;
    });

    onMount(async () => {
        await loadFiles();
    });

    async function loadFiles() {
        isLoading = true;
        try {
            allFiles = await ListFiles();
            console.log("Files loaded:", allFiles);
        } catch (e) {
            console.error(e);
            alert("Ошибка загрузки файлов: " + e);
        } finally {
            isLoading = false;
        }
    }

    async function deleteChapter(group) {
        if (!confirm(`Удалить ${group.files.length} файлов из главы "${group.date}"?`)) return;

        const filenames = group.files.map(f => f.name);
        try {
            await DeleteFiles(filenames);
            // Optimistic update
            allFiles = allFiles.filter(f => !filenames.includes(f.name));
        } catch (e) {
            alert("Ошибка удаления: " + e);
        }
    }

    let expandedGroups = $state(new Set());

    function toggleGroup(id) {
        if (expandedGroups.has(id)) {
            expandedGroups.delete(id);
        } else {
            expandedGroups.add(id);
        }
        // Force reactivity
        expandedGroups = new Set(expandedGroups);
    }

</script>

<div class="storage-container">
    <div class="header">
        <h2>Менеджер Хранилища</h2>
        <div class="controls">
            <label>
                Группировка: {sliderValue} мин
                <!-- Optimization: Update logic only on change (mouse up), update visual on input -->
                <input 
                    type="range" 
                    min="1" 
                    max="60" 
                    value={sliderValue} 
                    oninput={(e) => sliderValue = parseInt(e.currentTarget.value)} 
                    onchange={() => timeThreshold = sliderValue} 
                />
            </label>
            <div class="stats">
                <span>Файлов: {allFiles.length}</span>
                <span>Глав: {groupedChapters.length}</span>
            </div>
            <Button onclick={loadFiles} variant="outlined">Обновить</Button>
        </div>
    </div>

    {#if isLoading}
        <div class="loading">Загрузка...</div>
    {:else}
        <div class="chapters-list">
            {#each groupedChapters as group (group.id)}
                {@const isExpanded = expandedGroups.has(group.id)}
                <div class="chapter-card">
                    <div class="card-header">
                        <div class="cover">
                            {#if group.files.length > 0}
                                <img src={group.files[0].url} alt="cover" decoding="async" />
                            {/if}
                        </div>
                        <div class="info">
                            <div class="date">{group.date}</div>
                            <div class="count">{group.files.length} стр.</div>
                        </div>
                        <div class="actions">
                            <Button variant="text" onclick={() => deleteChapter(group)}>
                                <Icon icon={iconDelete} />
                            </Button>
                            <Button variant="text" onclick={() => toggleGroup(group.id)}>
                                <Icon icon={isExpanded ? iconCollapse : iconExpand} />
                            </Button>
                        </div>
                    </div>
                    
                    {#if isExpanded}
                        <div class="card-body">
                            {#each group.files as file}
                                <div class="file-item">
                                    <img src={file.url} alt={file.name} loading="lazy" decoding="async" />
                                    <span class="filename" title={file.name}>{file.name.split('/').pop()}</span>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .storage-container {
        padding: 16px;
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 16px;
        box-sizing: border-box; /* Ensure padding is included in height */
    }

    .header {
        flex-shrink: 0; /* Prevent header from shrinking */
        background-color: var(--m3c-surface-container);
        padding: 16px;
        border-radius: 16px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 16px;
    }

    /* ... controls, stats, label styles ... */

    .chapters-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
        overflow-y: auto;
        flex-grow: 1; /* Take remaining space */
        min-height: 0; /* Allow shrinking */
        padding-bottom: 16px; /* Bottom spacing for scroll */
    }

    .chapter-card {
        background-color: var(--m3c-surface-container-low);
        border-radius: 12px;
        border: 1px solid var(--m3c-outline-variant);
        /* Optimization: Content Visibility */
        content-visibility: auto;
        contain-intrinsic-size: 1px 80px; /* Estimate header height */
    }

    .card-header {
        display: flex;
        align-items: center;
        padding: 8px 16px;
        gap: 16px;
    }

    .cover img {
        width: 48px;
        height: 48px;
        object-fit: cover;
        border-radius: 4px;
        background-color: var(--m3c-surface-variant);
    }

    .info {
        flex-grow: 1;
        display: flex;
        flex-direction: column;
    }

    .date {
        font-weight: 500;
        color: var(--m3c-on-surface);
    }

    .count {
        font-size: 0.8rem;
        background-color: var(--m3c-secondary-container);
        color: var(--m3c-on-secondary-container);
        padding: 2px 8px;
        border-radius: 12px;
        width: fit-content;
        margin-top: 4px;
    }

    .card-body {
        padding: 16px;
        background-color: var(--m3c-surface);
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
        gap: 8px;
        border-top: 1px solid var(--m3c-outline-variant);
    }

    .file-item {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .file-item img {
        width: 100%;
        aspect-ratio: 2/3;
        object-fit: cover;
        border-radius: 4px;
        background-color: var(--m3c-surface-variant);
    }

    .filename {
        font-size: 0.7rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        color: var(--m3c-on-surface-variant);
    }

    .loading {
        text-align: center;
        padding: 20px;
        color: var(--m3c-on-surface-variant);
    }
</style>
