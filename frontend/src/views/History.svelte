<script>
    import { onMount } from "svelte";

    import { Button, Card, Icon, snackbar, Snackbar } from "m3-svelte";
    import iconOpen from "@ktibow/iconset-material-symbols/open-in-new";
    import iconView from "@ktibow/iconset-material-symbols/visibility-outline";
    import iconCopy from "@ktibow/iconset-material-symbols/content-copy-outline";
    import editIcon from "@ktibow/iconset-material-symbols/edit-outline";
    import iconShare from "@ktibow/iconset-material-symbols/share-outline";
    import { GetHistory, ClearHistory } from "../../wailsjs/go/main/App";
    import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

    function copyLink(url) {
        navigator.clipboard.writeText(url);
        snackbar("Ссылка скопирована!", undefined, true);
    }

    let historyItems = [];
    onMount(async () => {
        historyItems = await GetHistory(50, 0);
    });

    async function getArticleviews(history_item) {
        const slug = history_item.url.split("/").pop();

        let url = `https://api.telegra.ph/getViews/${slug}`;
        const response = await fetch(url);
        const data = await response.json();
        if (data.error) {
            console.error("Error fetching views:", data.error);
            return 0;
        }
        return data.result.views;
    }
</script>

<Snackbar />
<div class="cards">
    {#each historyItems as item}
        <Card variant="filled">
            <div class="card-wrapper">
                <Button variant="text" onclick={() => BrowserOpenURL(item.url)}>
                    <div class="title">
                        <span>{item.title}</span>
                    </div>
                    <Icon icon={iconOpen} />
                </Button>
                <div class="data">{item.date}</div>
                <div class="views">
                    <Icon icon={iconView} />
                    <span>
                        {#await getArticleviews(item) then views}{views}{/await
                        }</span>
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
                    <Button disabled>
                        <Icon icon={editIcon} />
                        Редактировать
                    </Button>
                    <Button disabled>
                        <Icon icon={iconShare} />
                        Опубликовать
                    </Button>
                </div>
            </div>
        </Card>
    {/each}
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
</style>
