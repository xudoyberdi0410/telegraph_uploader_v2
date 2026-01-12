<script>
    import { onMount } from "svelte";
    import { Button, Card, Icon, snackbar, Snackbar } from "m3-svelte";

    import iconOpen from "@ktibow/iconset-material-symbols/open-in-new";
    import iconView from "@ktibow/iconset-material-symbols/visibility-outline";
    import iconCopy from "@ktibow/iconset-material-symbols/content-copy-outline";
    import editIcon from "@ktibow/iconset-material-symbols/edit-outline";
    import iconShare from "@ktibow/iconset-material-symbols/share-outline";

    import { GetHistory } from "../../wailsjs/go/main/App";
    import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
    import { navigationStore } from "../stores/navigation.svelte";
    import { editorStore } from "../stores/editor.svelte";

    let historyItems = $state([]);

    onMount(async () => {
        try {
            historyItems = await GetHistory(50, 0);
        } catch (e) {
            console.error("Ошибка загрузки истории:", e);
            snackbar("Не удалось загрузить историю");
        }
    });

    function copyLink(url) {
        navigator.clipboard.writeText(url);
        snackbar("Ссылка скопирована!", undefined, true);
    }

    async function getArticleViews(history_item) {
        if (!history_item.url) return 0;

        const parts = history_item.url.split("/").filter((p) => p);
        const slug = parts[parts.length - 1];

        try {
            const url = `https://api.telegra.ph/getViews/${slug}`;
            const response = await fetch(url);
            const data = await response.json();
            if (data.ok === false || data.error) {
                return 0;
            }

            return data.result?.views || 0;
        } catch (e) {
            return 0;
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return "";
        try {
            const date = new Date(dateStr);
            return new Intl.DateTimeFormat("ru-RU", {
                day: "numeric",
                month: "long",
                hour: "2-digit",
                minute: "2-digit",
            }).format(date);
        } catch {
            return dateStr;
        }
    }

    function publishAction(item) {
        navigationStore.navigateTo("telegram", {
            historyId: item.id,
            articleTitle: item.title,
            titleId: item.title_id || 0,
        });
    }
</script>

<Snackbar />
<div class="cards">
    {#each historyItems as item (item.id)}
        <Card variant="filled">
            <div class="card-wrapper">
                <Button variant="text" onclick={() => BrowserOpenURL(item.url)}>
                    <div class="title">
                        <span>{item.title}</span>
                    </div>
                    <Icon icon={iconOpen} />
                </Button>
                <div class="data">{formatDate(item.date)}</div>
                <div class="views">
                    <Icon icon={iconView} />
                    {#await getArticleViews(item)}
                        <span>...</span>
                    {:then views}
                        <span>{views}</span>
                    {:catch}
                        <span>0</span>
                    {/await}
                </div>
                <div class="actions">
                    <Button onclick={() => BrowserOpenURL(item.url)}>
                        <Icon icon={iconOpen} />
                        Открыть
                    </Button>
                    <Button onclick={() => copyLink(item.url)}>
                        <Icon icon={iconCopy} />
                        Копировать
                    </Button>
                    <Button onclick={() => editorStore.loadArticle(item)}>
                        <Icon icon={editIcon} />
                        Редактировать
                    </Button>
                    <Button onclick={() => publishAction(item)}>
                        <Icon icon={iconShare} />
                        Опубликовать
                    </Button>
                </div>
            </div>
        </Card>
    {/each}
    {#if historyItems.length === 0}
        <div class="empty-state">История пуста</div>
    {/if}
</div>

<style>
    .cards {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .card-wrapper {
        display: flex;
        flex-direction: column;
        gap: 8px;
        align-items: flex-start;
    }

    .title {
        display: flex;
        align-items: center;
        gap: 4px;
        font-weight: bold;
        font-size: 1.3em;
    }
    .data {
        font-size: small;
    }
    .views {
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .actions {
        margin-top: 10px;
        display: flex;
        width: 100%;
        gap: 10px;
    }
    .empty-state {
        text-align: center;
        padding: 40px;
        color: var(--m3c-on-surface-variant);
    }
</style>
