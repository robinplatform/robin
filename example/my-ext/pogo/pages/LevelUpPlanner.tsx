import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { useSelectOption } from '../components/EditableField';
import { ScrollWindow } from '../components/ScrollWindow';
import { SelectPage } from '../components/SelectPage';
import { megaCostForTime, MegaWaitDays, MegaWaitTime } from '../domain-utils';
import { HOUR_MS } from '../math';
import { fetchDbRpc } from '../server/db.server';

export function LevelUpPlanner() {
	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />
			</div>
		</div>
	);
}
