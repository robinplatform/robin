import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { useSelectOption } from '../components/EditableField';
import { ScrollWindow } from '../components/ScrollWindow';
import { SelectPage } from '../components/SelectPage';
import { megaCostForTime, MegaWaitDays, MegaWaitTime } from '../domain-utils';
import { HOUR_MS } from '../math';
import { fetchDbRpc } from '../server/db.server';

export function Day({ date }: { date: Date }) {
	return (
		<div
			style={{
				position: 'relative',
				height: '2rem',
				width: '2rem',
				borderRadius: '2rem',
				backgroundColor: 'blue',
			}}
		>
			<div
				style={{
					position: 'absolute',
					right: '2.5rem',
					top: '0',
					bottom: '0',
					width: '8rem',
					display: 'flex',
					flexDirection: 'column',
					justifyContent: 'center',
					textAlign: 'right',
				}}
			>
				{date.toDateString()}
			</div>
		</div>
	);
}

export function LevelUpPlanner() {
	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />
			</div>

			<ScrollWindow
				className={'full'}
				innerClassName={'col'}
				innerStyle={{ alignItems: 'center' }}
			>
				<Day date={new Date()} />
			</ScrollWindow>
		</div>
	);
}
