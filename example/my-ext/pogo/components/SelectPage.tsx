import { create } from 'zustand';
import React from 'react';

const DefaultPage = 'levelup' as const;

// I'm not handling errors in this file, because... oh well. Whatever. Meh.
const PageTypes = ['pokemon', 'planner', 'tables', 'levelup'] as const;
type PageType = typeof PageTypes[number];
export const useCurrentPage = create<{
	page: PageType;
	setPage: (a: PageType) => void;
}>((set, get) => {
	return {
		setPage: (a) => set({ page: a }),
		page: DefaultPage,
	};
});

export function SelectPage() {
	const { page, setPage } = useCurrentPage();

	return (
		<select
			value={page}
			onChange={(evt) => setPage(evt.target.value as PageType)}
		>
			{PageTypes.map((page) => (
				<option key={page} value={page}>
					{page}
				</option>
			))}
		</select>
	);
}
