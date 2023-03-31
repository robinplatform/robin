import React from 'react';
import { useCurrentPage } from './components/SelectPage';
import { EventPlanner } from './pages/EventPlanner';
import { PokemonManager } from './pages/PokemonManager';
import { CostTables } from './pages/CostTables';
import { LevelUpPlanner } from './pages/LevelUpPlanner';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import { fetchDbRpc } from './server/db.server';
import { z } from 'zod';
import { useTopicQuery } from '@robinplatform/toolkit/react/stream';

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
export function Pogo(): JSX.Element {
	const { page } = useCurrentPage();

	const { refetch } = useRpcQuery(fetchDbRpc, {});
	useTopicQuery({
		topicId: {
			category: '/app-topics/my-ext/pogo',
			key: 'db',
		},
		resultType: z.object({}),
		fetchState: () => Promise.resolve({ state: 0, counter: 0 }),
		reducer: (a, b) => {
			refetch();
			return a;
		},
	});

	switch (page) {
		case 'pokemon':
			return <PokemonManager />;
		case 'planner':
			return <EventPlanner />;
		case 'tables':
			return <CostTables />;
		case 'levelup':
			return <LevelUpPlanner />;
	}
}
