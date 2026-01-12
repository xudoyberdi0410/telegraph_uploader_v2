class NavigationStore {
    currentPage = $state("home");
    pageProps = $state({});

    navigateTo(page, props = {}) {
        this.currentPage = page;
        this.pageProps = props;
    }
}

export const navigationStore = new NavigationStore();
