type Props = {
	id: string;
};

export function AppWindow({ id }: Props) {
	return (
		<div className={'full col'}>
			<iframe
				className={''}
				src={`http://localhost:9010/app-resources/${id}/base.html`}
				style={{ border: '0', flexGrow: 1 }}
			></iframe>
		</div>
	);
}
