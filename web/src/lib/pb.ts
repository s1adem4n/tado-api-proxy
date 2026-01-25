import PocketBase, { type RecordService } from 'pocketbase';

export interface Base {
	id: string;
	created: string;
	updated: string;
}

export interface Account extends Base {
	tadoID: string;
	email: string;
	password: string;
	homes: string[];
}

export interface Client extends Base {
	clientID: string;
	redirectURI: string;
	scope: string;
	name: string;
	type: 'passwordGrant' | 'deviceCode';
}

export interface Code extends Base {
	client: string;
	token: string;
	deviceCode: string;
	userCode: string;
	verificationURI: string;
	expires: string;
}

export interface Home extends Base {
	tadoID: string;
	name: string;
}

export interface Requests extends Base {
	token: string;
	method: string;
	url: string;
	status: number;
}

export interface Token extends Base {
	account: string;
	client: string;
	status: 'valid' | 'invalid';
	accessToken: string;
	refreshToken: string;
	expires: string;
	used: string;
}

export interface TypedPocketBase extends PocketBase {
	collection(idOrName: string): RecordService;
	collection(idOrName: 'accounts'): RecordService<Account>;
	collection(idOrName: 'clients'): RecordService<Client>;
	collection(idOrName: 'codes'): RecordService<Code>;
	collection(idOrName: 'homes'): RecordService<Home>;
	collection(idOrName: 'requests'): RecordService<Requests>;
	collection(idOrName: 'tokens'): RecordService<Token>;
}

export const pb = new PocketBase() as TypedPocketBase;
