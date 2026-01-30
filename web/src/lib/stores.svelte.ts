import { pb, type Base } from '@/lib/pb';
import { type AuthRecord, type RecordService } from 'pocketbase';

export class NavigationStore {
	path: string = $state(window.location.pathname);
	query: Record<string, string> = $state(this.parseQuery());

	constructor() {
		$effect.root(() => {
			$effect(() => {
				const onPopState = () => {
					this.path = window.location.pathname;
					this.query = this.parseQuery();
				};
				window.addEventListener('popstate', onPopState);

				return () => window.removeEventListener('popstate', onPopState);
			});
		});
	}

	private parseQuery(): Record<string, string> {
		const params = new URLSearchParams(window.location.search);
		const result: Record<string, string> = {};
		params.forEach((value, key) => {
			result[key] = value;
		});
		return result;
	}

	navigate(path: string, query?: Record<string, string>) {
		const url = new URL(path, window.location.origin);
		if (query) {
			Object.entries(query).forEach(([key, value]) => {
				url.searchParams.set(key, value);
			});
		}
		window.history.pushState({}, '', url.pathname + url.search);
		this.path = url.pathname;
		this.query = query ?? {};
	}

	setQuery(key: string, value: string) {
		const url = new URL(window.location.href);
		url.searchParams.set(key, value);
		window.history.replaceState({}, '', url.pathname + url.search);
		this.query = { ...this.query, [key]: value };
	}

	getQuery(key: string, defaultValue?: string): string {
		return this.query[key] ?? defaultValue ?? '';
	}
}

export const navigation = new NavigationStore();

export class MultipleSubscription<T extends Base> {
	items: T[] = $state([]);
	promise: Promise<void> | null = null;

	constructor(service: RecordService<T>, filter?: () => string, transform?: (items: T[]) => T[]) {
		$effect(() => {
			this.promise = service.getFullList({ filter: filter?.() }).then((items) => {
				this.items = transform ? transform(items) : items;
			});
		});

		$effect(() => {
			const unsubscribe = service.subscribe('*', (e) => {
				let updated: T[] = [];
				switch (e.action) {
					case 'create':
						updated = [...this.items, e.record];
						this.items = transform ? transform(updated) : updated;
						break;
					case 'update':
						updated = this.items.map((item) => (item.id === e.record.id ? e.record : item));
						this.items = transform ? transform(updated) : updated;
						break;
					case 'delete':
						updated = this.items.filter((item) => item.id !== e.record.id);
						this.items = transform ? transform(updated) : updated;
						break;
				}
			});

			return () => unsubscribe.then((fn) => fn());
		});
	}
}

export class SingleSubscription<T extends Base> {
	item: T | null = $state(null);
	promise: Promise<void> | null = null;

	constructor(service: RecordService<T>, id: () => string) {
		$effect(() => {
			this.promise = service.getOne(id()).then((item) => {
				this.item = item;
			});
		});

		$effect(() => {
			const unsubscribe = service.subscribe(id(), (e) => {
				switch (e.action) {
					case 'update':
						this.item = e.record;
						break;
					case 'delete':
						this.item = null;
						break;
				}
			});

			return () => unsubscribe.then((fn) => fn());
		});
	}
}

export class AuthStore {
	record: AuthRecord = $state(pb.authStore.record);
	isValid: boolean = $state(pb.authStore.isValid);

	constructor() {
		$effect(() => {
			const unsubscribe = pb.authStore.onChange(() => {
				this.record = pb.authStore.record;
				this.isValid = pb.authStore.isValid;
			});

			return () => unsubscribe();
		});
	}
}
