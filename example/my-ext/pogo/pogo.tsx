import React from 'react';
import { useCurrentPage } from './components/SelectPage';
import { PokemonManager } from './pages/PokemonManager';

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
export function Pogo() {
	const { page } = useCurrentPage();

	switch (page) {
		case 'pokemon':
			return <PokemonManager />;
	}
}
