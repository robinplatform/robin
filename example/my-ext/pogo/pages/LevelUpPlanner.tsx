import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { useSelectOption } from '../components/EditableField';
import { ScrollWindow } from '../components/ScrollWindow';
import { SelectPage } from '../components/SelectPage';
import { megaCostForTime, MegaWaitDays, MegaWaitTime } from '../domain-utils';
import { arrayOfN, DAY_MS, HOUR_MS } from '../math';
import { fetchDbRpc } from '../server/db.server';

// Include cancel or not
// Specify locks/planned actions

function DateText({ date }: { date: Date }) {
	return (
		<div
			style={{
				position: 'absolute',
				right: '2rem',
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
	);
}

function SmallDot() {
	return (
		<div
			style={{
				position: 'absolute',
				top: '0',
				bottom: '0',
				height: '1rem',
				width: '1rem',
				borderRadius: '1rem',
				backgroundColor: 'blue',
			}}
		/>
	);
}

function BigDot() {
	return (
		<div
			style={{
				position: 'absolute',
				top: '-0.5rem',
				left: '-0.5rem',
				height: '2rem',
				width: '2rem',
				borderRadius: '2rem',
				backgroundColor: 'blue',
			}}
		/>
	);
}

function DayBox({ children }: { children: React.ReactNode }) {
	return (
		<div
			style={{
				position: 'relative',
				height: '1rem',
				width: '1rem',
				borderRadius: '1rem',
				backgroundColor: 'blue',
			}}
		>
			{children}
		</div>
	);
}

export function LevelUpPlanner() {
	const days = arrayOfN(10).map((i) => new Date(Date.now() + (i - 4) * DAY_MS));

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />
			</div>

			<ScrollWindow
				className={'full'}
				innerClassName={'col robin-pad'}
				innerStyle={{
					alignItems: 'center',
					background: 'white',
					gap: '1.5rem',
				}}
			>
				{days.map((d) => (
					<DayBox>
						{d.toDateString() === new Date().toDateString() ? (
							<BigDot />
						) : (
							<SmallDot />
						)}
						<DateText date={d} />
					</DayBox>
				))}
			</ScrollWindow>
		</div>
	);
}
