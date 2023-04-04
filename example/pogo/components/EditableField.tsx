import React from 'react';
import '@robinplatform/toolkit/styles.css';
import './timer.scss';

type EditFieldProps<T> = {
	value: T;
	setValue: (t: T) => void;
	parseFunc: (t: string) => T | undefined;
	disabled?: boolean;
	children?: React.ReactNode;
};

export function EditField<T>({
	value,
	setValue,
	disabled,
	parseFunc,
	children,
}: EditFieldProps<T>) {
	const [editing, setEditing] = React.useState<boolean>(false);
	const [valueState, setValueState] = React.useState<string>(`${value}`);

	React.useEffect(() => {
		setValueState(`${value}`);
	}, [value]);

	const parseFuncRef = React.useRef(parseFunc);
	parseFuncRef.current = parseFunc;

	const valueParsed = React.useMemo(
		() => parseFuncRef.current(valueState),
		[valueState],
	);

	return (
		<div className={'row'} style={{ gap: '0.25rem' }}>
			<div className={'row'} style={{ position: 'relative', padding: '4px' }}>
				<div
					style={{
						position: 'absolute',
						left: 0,
						top: 0,
						bottom: 0,
						right: 0,
					}}
				>
					<input
						type="text"
						value={valueState}
						onChange={(evt) => setValueState(evt.target.value)}
						style={{
							display: editing ? undefined : 'none',
							padding: '2px',
							border: '2px solid gray',
							height: '100%',
							width: '100%',
						}}
					/>
				</div>

				{/* We use `visibility` here to ensure that layout still happens, so that
					the box doesn't change shape during editing, but that the
					stuff underneath doesn't overlap visually in the process.
				 */}
				<div style={{ visibility: editing ? 'hidden' : undefined }}>
					{children}
				</div>
			</div>

			<button
				disabled={disabled || valueParsed === undefined}
				style={{ fontSize: '0.75rem', width: '2.3rem', textAlign: 'center' }}
				onClick={() => {
					if (editing) {
						if (valueParsed === undefined) {
							return;
						}

						setValue(valueParsed);
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

export function useSelectOption<T>(options: Partial<Record<number, T>>) {
	const [selectedIndex, setSelected] = React.useState<number>(NaN);

	return {
		selected: options[selectedIndex],
		value: `${selectedIndex}`,
		onChange: (evt: React.ChangeEvent<HTMLSelectElement>) =>
			setSelected(Number.parseInt(evt.target.value)),
	};
}
