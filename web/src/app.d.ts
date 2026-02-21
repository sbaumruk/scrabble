// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
	namespace App {}

	interface Window {
		_paq?: Array<Array<string | number>>;
	}
}

export {};
