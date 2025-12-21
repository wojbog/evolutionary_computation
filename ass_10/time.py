import pandas as pd

def read_csv_file(file_path):
    data = pd.read_csv(file_path)
    return data

LETTER = "B"


file_path = f'res{LETTER}.csv'

data = read_csv_file(file_path)
print(data.head())


# for each method calculate average time duration_ms
summary = (
    data.groupby('method')['duration_s']
    .agg(min_time='min', max_time='max', mean_time='mean')
    .reset_index()
)

# round numeric columns for nicer display
for col in ['min_time', 'max_time', 'mean_time']:
    summary[col] = summary[col].round(3)
print(summary.to_string(index=False))
# optionally save the prepared table to CSV
summary.to_csv(f'summary_time_by_method_{LETTER}.csv', index=False)

