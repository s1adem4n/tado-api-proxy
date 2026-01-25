import { pb, type Base } from '@/lib/pb';
import { type AuthRecord, type RecordService } from 'pocketbase';

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
