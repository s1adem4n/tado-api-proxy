import { pb, type Base } from '@/lib/pb';
import { type AuthRecord, type RecordService } from 'pocketbase';

export class MultipleSubscription<T extends Base> {
	items: T[] = $state([]);
	promise: Promise<void> | null = null;

	constructor(service: RecordService<T>, filter: () => string, transform: (items: T[]) => T[]) {
		$effect(() => {
			this.promise = service.getFullList({ filter: filter() }).then((items) => {
				this.items = transform(items);
			});
		});

		$effect(() => {
			const unsubscribe = service.subscribe('*', (e) => {
				switch (e.action) {
					case 'create':
						this.items = transform([...this.items, e.record]);
						break;
					case 'update':
						this.items = this.items.map((item) => (item.id === e.record.id ? e.record : item));
						break;
					case 'delete':
						this.items = this.items.filter((item) => item.id !== e.record.id);
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
