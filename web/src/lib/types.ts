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
