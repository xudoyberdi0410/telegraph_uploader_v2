<script>
    import { onMount } from "svelte";
    import { GetTelegramUser } from "../../wailsjs/go/main/App";

    let user = $state(null);
    let isLoading = $state(true);

    onMount(async () => {
        try {
            user = await GetTelegramUser();
        } catch (e) {
            console.error("Failed to load user:", e);
        } finally {
            isLoading = false;
        }
    });

    function getInitials(u) {
        if (!u) return "?";
        let i = "";
        if (u.first_name) i += u.first_name[0];
        if (u.last_name) i += u.last_name[0];
        return i.toUpperCase();
    }
</script>

<div class="dashboard-container">
    {#if isLoading}
        <div class="loading">Loading...</div>
    {:else if user}
        <div class="profile-card">
            <div class="avatar-circle">
                {#if user.photo}
                    <!-- If we had photo logic, it would go here. For now initials. -->
                    <img
                        src={"data:image/jpeg;base64," + user.photo}
                        alt="Avatar"
                    />
                {:else}
                    <span class="initials">{getInitials(user)}</span>
                {/if}
            </div>
            <div class="user-info">
                <h2>{user.first_name} {user.last_name || ""}</h2>
                {#if user.username}<p>@{user.username}</p>{/if}
            </div>
        </div>

        <div class="instructions">
            <p>Выберите статью в разделе "История", чтобы опубликовать её.</p>
        </div>
    {:else}
        <div class="error">Не удалось загрузить профиль.</div>
    {/if}
</div>

<style>
    .dashboard-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--m3-sys-color-on-surface);
    }

    .profile-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 16px;
        margin-bottom: 32px;
    }

    .avatar-circle {
        width: 120px;
        height: 120px;
        border-radius: 50%;
        background-color: var(--m3-sys-color-primary-container);
        color: var(--m3-sys-color-on-primary-container);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 48px;
        font-weight: bold;
        overflow: hidden;
    }

    .avatar-circle img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .user-info h2 {
        margin: 0;
        font-size: 24px;
        font-weight: 500;
    }

    .user-info p {
        margin: 4px 0 0 0;
        color: var(--m3-sys-color-on-surface-variant);
        font-size: 16px;
    }

    .instructions {
        padding: 16px;
        background-color: var(--m3-sys-color-surface-variant);
        color: var(--m3-sys-color-on-surface-variant);
        border-radius: 12px;
        text-align: center;
        max-width: 400px;
    }
</style>
