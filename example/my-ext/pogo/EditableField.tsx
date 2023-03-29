import React from 'react';
import '@robinplatform/toolkit/styles.css';
import './timer.scss';

export function EditableInt({
	value,
	setValue,
	disabled,
}: {
	disabled?: boolean;
	value: number;
	setValue: (n: number) => void;
}) {
	const [editing, setEditing] = React.useState<boolean>(false);
	const [valueState, setValueState] = React.useState<string>(`${value}`);

	React.useEffect(() => {
		setValueState(`${value}`);
	}, [value]);

	return (
		<div
			className={'row'}
			style={{
				height: '1.5rem',
				gap: '1rem',
			}}
		>
			{editing ? (
				<input
					style={{ width: '3rem', textAlign: 'right' }}
					value={valueState}
					onChange={(evt) => setValueState(evt.target.value)}
				/>
			) : (
				<p style={{ width: '3rem', textAlign: 'right' }}>{value}</p>
			)}

			<button
				disabled={disabled || Number.isNaN(Number.parseInt(valueState))}
				style={{ fontSize: '0.75rem' }}
				onClick={() => {
					if (editing) {
						setValue(Number.parseInt(valueState));
						setEditing(false);
					} else {
						setEditing(true);
					}
				}}
			>
				{editing ? 'Done' : 'Edit'}
			</button>
		</div>
	);
}
