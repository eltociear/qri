# Set a body containing a config field and a secret
def transform(ds, ctx):
  prev_body = ds.get_body()
  ds.set_body([['Name', ctx.get_config('animal_name')],
               ['Sound', ctx.get_secret('animal_sound')]])
