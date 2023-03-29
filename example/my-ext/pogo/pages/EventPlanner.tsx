import React from 'react';
import { SelectPage } from '../components/SelectPage';

export function EventPlanner() {
	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />
			</div>
		</div>
	);
}
