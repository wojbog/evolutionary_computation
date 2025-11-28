import pandas as pd

def read_csv_file(file_path):
    data = pd.read_csv(file_path)
    return data

LETTER = "B"
file_path = f'result{LETTER}.csv'


data = read_csv_file(file_path)
print(data.head())


# for each method calculate average time duration_ms
summary = (
    data.groupby('method')['objective']
    .agg(min_obj='min', max_obj='max', mean_obj='mean')
    .reset_index()
)

# round numeric columns for nicer display
for col in ['min_obj', 'max_obj', 'mean_obj']:
    summary[col] = summary[col].round(3)
print(summary.to_string(index=False))
# optionally save the prepared table to CSV
summary.to_csv(f'summary_obj_{LETTER}.csv', index=False)
