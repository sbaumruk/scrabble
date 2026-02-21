export interface Move {
	x: number;
	y: number;
	dir: 'H' | 'V';
	tiles: string;
	word: string;
	score: number;
	newPositions: [number, number][];
}

export interface Ruleset {
	name: string;
	bingoBonus: number;
	letterPoints: Record<string, number>;
	tripleWord: [number, number][];
	doubleWord: [number, number][];
	tripleLetter: [number, number][];
	doubleLetter: [number, number][];
}

export interface BoardMeta {
	id: string;
	name: string;
	createdAt: string;
	updatedAt: string;
}

export interface BoardRecord extends BoardMeta {
	board: string[];
	isOwner?: boolean;
}
