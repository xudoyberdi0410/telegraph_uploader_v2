<script>
    import { onMount, createEventDispatcher } from "svelte";
    import { GetHistory, ClearHistory } from "../../wailsjs/go/main/App";
    import { fly } from "svelte/transition";

    const dispatch = createEventDispatcher();
    let historyItems = [];

    onMount(async () => {
        historyItems = await GetHistory(50, 0);
    });

    async function handleClear() {
        if (confirm("Удалить всю историю?")) {
            await ClearHistory();
            historyItems = [];
        }
    }

    // Форматирование даты
    function formatDate(dateStr) {
        return new Date(dateStr).toLocaleString("ru-RU");
    }

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

<div class="history-modal" transition:fly={{ y: 20, duration: 300 }}>
    <div class="header">
        <h3>История загрузок</h3>
        <div class="actions">
            <button class="btn-close" on:click={() => dispatch("close")}
                >✕</button
            >
        </div>
    </div>

    <div class="list">
        {#each historyItems as item}
            <div class="item">
                <div class="info">
                    <span class="date">{item.date}</span>
                    <span class="title">{item.title}</span>
                    <span class="count">{item.img_count} стр.</span>
                    <span class="count"
                        >Просмотры: {#await getArticleviews(item) then views}{views}{/await}</span
                    >
                </div>
                <div
                    class="link"
                    on:click={() => navigator.clipboard.writeText(item.url)}
                >
                    Копировать
                </div>
            </div>
        {:else}
            <div class="empty">История пуста</div>
        {/each}
    </div>
</div>

<style>
    .history-modal {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 500px;
        height: 600px;
        background: #2a2a2a;
        border: 1px solid #444;
        border-radius: 8px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
        z-index: 11;
        display: flex;
        flex-direction: column;
        color: #eee;
    }
    .header {
        padding: 15px;
        border-bottom: 1px solid #444;
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: #333;
        border-radius: 8px 8px 0 0;
    }
    .list {
        flex: 1;
        overflow-y: auto;
        padding: 10px;
    }
    .item {
        background: #1f1f1f;
        padding: 10px;
        margin-bottom: 8px;
        border-radius: 4px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .info {
        display: flex;
        flex-direction: column;
        gap: 2px;
        text-align: left;
    }
    .date {
        font-size: 0.75rem;
        color: #888;
    }
    .title {
        font-weight: bold;
        font-size: 0.95rem;
    }
    .count {
        font-size: 0.8rem;
        color: #aaa;
    }
    .link {
        color: #4a90e2;
        text-decoration: none;
        padding: 5px 10px;
        border: 1px solid #4a90e2;
        border-radius: 4px;
        cursor: pointer;
    }
    .link:hover {
        background: #4a90e2;
        color: white;
    }
    .btn-close {
        background: none;
        border: none;
        color: #fff;
        font-size: 1.2rem;
        cursor: pointer;
    }
    .btn-clear {
        background: #5a2a2a;
        color: #ffaaaa;
        border: none;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
        margin-right: 10px;
    }
    .empty {
        padding: 20px;
        color: #666;
    }
</style>
